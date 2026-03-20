package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

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
	log.Printf("Saving unittestTask to DB...")
	dbStart := time.Now()
	err := s.unittestStore.Save(ctx, task)
	log.Printf("UnittestTask save to DB took %v", time.Since(dbStart))
	if err != nil {
		return 0, err
	}

	// 异步起动一个协程，处理 unittest 任务
	go func() {
		// 使用一个独立的 context，避免因为主请求结束导致协程被取消
		ctx := context.Background()

		// 1、根据NormalCoverDataPartitionKey，获取归一化的行覆盖率数据，以及diff数据。
		pk := partitionkey.NewUnitTestNormalCovKey(task.Module, task.Branch, task.Commit, task.RunID)
		if len(task.NormalCoverDataPartitionKey) < 1000 {
			log.Printf("Unmarshaling NormalCoverDataPartitionKey: %s", task.NormalCoverDataPartitionKey)
		} else {
			log.Printf("Unmarshaling large NormalCoverDataPartitionKey (length: %d)", len(task.NormalCoverDataPartitionKey))
		}
		err := pk.Unmarshal(task.NormalCoverDataPartitionKey)
		if err != nil {
			log.Printf("unmarshal normal cover data partition key failed: %v, input: %s", err, task.NormalCoverDataPartitionKey)
			return
		}
		log.Printf("Unmarshaled PK path prefix: %s", pk.RealPathPrefix())

		// 获取归一化的行覆盖率数据 .json
		jsonPath := pk.RealPathPrefix() + ".json"
		log.Printf("Fetching JSON coverage data from OSS: %s in bucket %s", jsonPath, s.bucketName)
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
			Module:     task.Module, // 使用请求中的 module，不裁剪
			Branch:     covNormalInfo.Branch,
			Commit:     covNormalInfo.Commit,
			BaseCommit: covNormalInfo.BaseCommit,
		}

		// 修复路径问题：确定需要从路径中剥离的前缀
		stripPrefix := ""
		for path := range covNormalInfo.CoverageData {
			// 尝试在路径中寻找 module 名称
			// 例如 path 为 /.../github.com/shuaibizhang/transparent-context/demo/main.go
			// module 为 github.com/shuaibizhang/transparent-context
			if idx := strings.Index(path, reportMeta.Module); idx != -1 {
				stripPrefix = path[:idx+len(reportMeta.Module)+1] // +1 是为了去掉后面的 /
				break
			}

			// 如果没找到完整的 module，尝试找最后一节
			moduleParts := strings.Split(reportMeta.Module, "/")
			lastPart := moduleParts[len(moduleParts)-1]
			if idx := strings.Index(path, lastPart); idx != -1 {
				stripPrefix = path[:idx+len(lastPart)+1]
				break
			}
		}

		if stripPrefix != "" {
			log.Printf("Using module: %s, stripping path prefix: %s", reportMeta.Module, stripPrefix)
		}

		log.Printf("Starting to generate report for module: %s, branch: %s, commit: %s", reportMeta.Module, reportMeta.Branch, reportMeta.Commit)
		reportPk := partitionkey.NewReportKey(partitionkey.UnitTest, reportMeta.Module, reportMeta.Branch, reportMeta.Commit)
		rep, err := s.reportManager.CreateReport(ctx, reportMeta, reportPk)
		if err != nil {
			log.Printf("create report failed: %v", err)
			return
		}

		// 填充数据
		fileCount := 0
		for path, data := range covNormalInfo.CoverageData {
			fileCount++

			// 清理路径
			displayPath := path
			if stripPrefix != "" && strings.HasPrefix(path, stripPrefix) {
				displayPath = strings.TrimPrefix(path, stripPrefix)
			}

			// 特殊处理：如果路径中包含 coverage.out/，尝试去掉它
			// 这种情况通常是由于某些工具在生成覆盖率时引入了中间目录
			if strings.Contains(displayPath, "coverage.out/") {
				oldPath := displayPath
				displayPath = strings.ReplaceAll(displayPath, "coverage.out/", "")
				log.Printf("Stripped 'coverage.out/' from path: %s -> %s", oldPath, displayPath)
			}

			var fileDiffInfo report.FileDiffInfo
			if gitDiffMap != nil {
				// 注意：gitDiffMap 中的 path 可能是原始 path，也可能是清理后的 path，
				// 这里假设 gitDiffMap 对应的是原始上报的 path
				if dFile, ok := gitDiffMap.DiffFileMap[path]; ok {
					fileDiffInfo.AddLines = uint32(len(dFile.Hunks))
					for _, hunk := range dFile.Hunks {
						for _, line := range hunk.NewFileLines {
							fileDiffInfo.AddedLines = append(fileDiffInfo.AddedLines, uint32(line))
						}
					}
				}
			}

			err := rep.AddFile(displayPath, data.CoverLineData, fileDiffInfo)
			if err != nil {
				log.Printf("add file %s to report failed: %v", displayPath, err)
				continue
			}
		}
		log.Printf("Added %d files to report", fileCount)

		// 持久化报告
		log.Printf("Flushing report to OSS...")
		err = rep.Flush(ctx)
		if err != nil {
			log.Printf("flush report failed: %v", err)
			return
		}
		log.Printf("Report flushed successfully")
		rep.Close(ctx)
		log.Printf("Report closed successfully")

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
