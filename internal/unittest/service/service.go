package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/shuaibizhang/codecoverage/internal/diff"
	"github.com/shuaibizhang/codecoverage/internal/oss"
	"github.com/shuaibizhang/codecoverage/internal/parser"
	"github.com/shuaibizhang/codecoverage/internal/reports/manager"
	"github.com/shuaibizhang/codecoverage/internal/reports/report"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/shuaibizhang/codecoverage/internal/unittest/store"
)

type UnitTestService interface {
	UploadUnittestReport(ctx context.Context, task *store.UnittestTask) (uint64, error)
}

type unitTestService struct {
	unittestStore store.UnitTestStore
	ossCli        oss.OSS
	reportManager manager.ReportManager
	bucketName    string
}

func NewUnitTestService(s store.UnitTestStore, ossCli oss.OSS, reportManager manager.ReportManager, bucketName string) UnitTestService {
	return &unitTestService{
		unittestStore: s,
		ossCli:        ossCli,
		reportManager: reportManager,
		bucketName:    bucketName,
	}
}

func (s *unitTestService) UploadUnittestReport(ctx context.Context, task *store.UnittestTask) (uint64, error) {
	if s.unittestStore == nil {
		return 0, fmt.Errorf("unittest store is not initialized")
	}
	// 保存到数据库中
	err := s.unittestStore.Save(ctx, task)
	if err != nil {
		return 0, err
	}

	// 异步起动一个协程，处理 unittest 任务
	go func() {
		// 使用一个独立的 context，避免因为主请求结束导致协程被取消
		ctx := context.Background()

		// 1、根据NormalCoverDataPartitionKey，获取归一化的行覆盖率数据，以及diff数据。
		pk := partitionkey.NewUnitTestNormalCovKey(task.Module, task.Branch, task.Commit, task.RunID)
		err := pk.Unmarshal(task.NormalCoverDataPartitionKey)
		if err != nil {
			log.Printf("unmarshal normal cover data partition key failed: %v", err)
			return
		}

		// 获取归一化的行覆盖率数据 .json
		jsonPath := pk.RealPathPrefix() + ".json"
		jsonReader, err := s.ossCli.GetObject(ctx, s.bucketName, jsonPath)
		if err != nil {
			log.Printf("get json from oss failed: %v, path: %s", err, jsonPath)
			return
		}
		defer jsonReader.Close()

		jsonData, err := io.ReadAll(jsonReader)
		if err != nil {
			log.Printf("read json data failed: %v", err)
			return
		}

		var covNormalInfo parser.CovNormalInfo
		err = json.Unmarshal(jsonData, &covNormalInfo)
		if err != nil {
			log.Printf("unmarshal cov normal info failed: %v", err)
			return
		}

		// 获取 diff 数据 .diff (可选)
		var gitDiffMap *diff.GitDiffMap
		diffPath := pk.RealPathPrefix() + ".diff"
		diffReader, err := s.ossCli.GetObject(ctx, s.bucketName, diffPath)
		if err == nil {
			defer diffReader.Close()
			diffData, err := io.ReadAll(diffReader)
			if err == nil {
				gitModel := diff.NewGitModel()
				gitDiff, err := gitModel.ParserGitDiffFile(string(diffData))
				if err == nil {
					gitDiffMap = gitDiff.CovertToMap()
				}
			}
		}

		// 2、生成对应的coverReport
		reportMeta := report.MetaInfo{
			Module:     covNormalInfo.Module,
			Branch:     covNormalInfo.Branch,
			Commit:     covNormalInfo.Commit,
			BaseCommit: covNormalInfo.BaseCommit,
		}

		reportPk := partitionkey.NewReportKey(partitionkey.UnitTest, reportMeta.Module, reportMeta.Branch, reportMeta.Commit)
		rep, err := s.reportManager.CreateReport(ctx, reportMeta, reportPk)
		if err != nil {
			log.Printf("create report failed: %v", err)
			return
		}

		// 填充数据
		for path, data := range covNormalInfo.CoverageData {
			var fileDiffInfo report.FileDiffInfo
			if gitDiffMap != nil {
				if dFile, ok := gitDiffMap.DiffFileMap[path]; ok {
					fileDiffInfo.AddLines = uint32(len(dFile.Hunks)) // 简便处理，这里可能需要更细致的行统计
					// 统计新增行号
					for _, hunk := range dFile.Hunks {
						for _, line := range hunk.NewFileLines {
							fileDiffInfo.AddedLines = append(fileDiffInfo.AddedLines, uint32(line))
						}
					}
				}
			}

			err := rep.AddFile(path, data.CoverLineData, fileDiffInfo)
			if err != nil {
				log.Printf("add file %s to report failed: %v", path, err)
				continue
			}
		}

		// 持久化报告
		err = rep.Flush(ctx)
		if err != nil {
			log.Printf("flush report failed: %v", err)
			return
		}
		rep.Close(ctx)

		// 3、修改unittestTask的状态为已处理，同时将获取的ReportPartitionKey保存到unittestTask中
		task.Status = "processed" // 已处理
		reportPkStr, _ := reportPk.Marshal()
		task.ReportPartitionKey = reportPkStr
		err = s.unittestStore.Save(ctx, task)
		if err != nil {
			log.Printf("update unittest task status failed: %v", err)
		}
	}()

	return task.ID, nil
}
