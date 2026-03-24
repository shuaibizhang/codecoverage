package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/coverage"
	"github.com/shuaibizhang/codecoverage/internal/diff"
	diffService "github.com/shuaibizhang/codecoverage/internal/diff/service"
	"github.com/shuaibizhang/codecoverage/internal/oss"
	"github.com/shuaibizhang/codecoverage/internal/parser"
	"github.com/shuaibizhang/codecoverage/internal/reports/manager"
	"github.com/shuaibizhang/codecoverage/internal/reports/report"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	sysStore "github.com/shuaibizhang/codecoverage/internal/systest/store"
	"github.com/shuaibizhang/codecoverage/logger"
	"github.com/shuaibizhang/codecoverage/store"
)

const TagSystestService = "SystestService"

type SystestService interface {
	UploadSystestCoverData(ctx context.Context, req *coverage.UploadSystestCoverDataRequest) (uint64, error)
}

type systestService struct {
	systestStore  sysStore.SystestStore
	ossCli        oss.OSS
	reportManager manager.ReportManager
	diffService   diffService.Service
	bucketName    string
	logger        logger.Logger
}

func NewSystestService(s sysStore.SystestStore, ossCli oss.OSS, reportManager manager.ReportManager, diffService diffService.Service, bucketName string) SystestService {
	return &systestService{
		systestStore:  s,
		ossCli:        ossCli,
		reportManager: reportManager,
		diffService:   diffService,
		bucketName:    bucketName,
		logger:        logger.Default(),
	}
}

// UploadSystestCoverData 上报系统测试任务元数据
func (s *systestService) UploadSystestCoverData(ctx context.Context, req *coverage.UploadSystestCoverDataRequest) (uint64, error) {
	s.logger.Infof(ctx, TagSystestService, "Entering UploadSystestCoverData")
	// 1、获取或创建任务
	task, err := s.getOrCreateTask(ctx, req)
	if err != nil {
		s.logger.Errorf(ctx, TagSystestService, "getOrCreateTask failed: %v", err)
		return 0, err
	}
	s.logger.Infof(ctx, TagSystestService, "Task created/found: %d", task.ID)

	// 2、启动后台处理
	s.logger.Infof(ctx, TagSystestService, "Starting goroutine for background processing")
	go func() {
		err := s.runBackgroundProcessing(task, req)
		if err != nil {
			s.logger.Errorf(ctx, TagSystestService, "Background processing failed: %v", err)
		}
	}()

	return task.ID, nil
}

// getOrCreateTask 获取并创建任务
func (s *systestService) getOrCreateTask(ctx context.Context, req *coverage.UploadSystestCoverDataRequest) (*sysStore.SystestTask, error) {
	task, err := s.systestStore.Query(ctx, req.Module, req.Branch, req.Commit)
	if err != nil && !errors.Is(err, store.ErrRecordNotFound) {
		return nil, fmt.Errorf("query systest task failed: %w", err)
	}

	if errors.Is(err, store.ErrRecordNotFound) {
		task = &sysStore.SystestTask{
			Language:                    req.Language,
			Module:                      req.Module,
			Branch:                      req.Branch,
			Commit:                      req.Commit,
			BaseCommit:                  req.BaseCommit,
			CommitCreateTime:            time.Now(), // 默认使用当前时间
			NormalCoverDataPartitionKey: req.NormalCoverDataPartitionKey,
			Status:                      "pending",
		}
		// 创建新报告分区key
		reportPk := partitionkey.NewSystestReportKey(task.Module, task.Branch, task.Commit)
		reportPkStr, _ := reportPk.Marshal()
		task.ReportPartitionKey = reportPkStr
		task.CreatedTime = time.Now()
	}

	// 填充其他字段
	if req.CreatedTime != "" {
		if t, errParse := time.Parse(time.RFC3339, req.CreatedTime); errParse == nil {
			task.CreatedTime = t
		} else if t, errParse := time.Parse("2006-01-02 15:04:05", req.CreatedTime); errParse == nil {
			task.CreatedTime = t
		}
	}

	// 保存任务元 metadata
	if err := s.systestStore.Save(ctx, task); err != nil {
		return nil, fmt.Errorf("save systest task failed: %w", err)
	}
	return task, nil
}

func (s *systestService) runBackgroundProcessing(task *sysStore.SystestTask, req *coverage.UploadSystestCoverDataRequest) error {
	ctx := context.Background()
	s.logger.Infof(ctx, TagSystestService, "Starting background processing for task %d (%s/%s/%s)", task.ID, task.Module, task.Branch, task.Commit)

	// 1. 获取归一化覆盖率数据
	var coverNormalInfo map[string]*coverage.CoverData
	if req.CoverData != nil {
		// 增量上报，直接使用请求中的数据
		coverNormalInfo = req.CoverData
	} else if task.NormalCoverDataPartitionKey != "" {
		// 全量上报，从 OSS 中获取基础归一化覆盖率数据
		info, _, err := s.fetchNormalInfo(ctx, task)
		if err != nil {
			s.logger.Errorf(ctx, TagSystestService, "fetchNormalInfo failed: %v", err)
			return err
		}
		// 转换类型
		coverNormalInfo = make(map[string]*coverage.CoverData)
		for path, data := range info.CoverageData {
			coverNormalInfo[path] = &coverage.CoverData{
				TotalLines:    uint32(data.TotalLines),
				CoverLines:    uint32(data.CoverLines),
				InstrLines:    uint32(data.InstrLines),
				CoverLineData: data.CoverLineData,
			}
		}
	}

	if coverNormalInfo == nil {
		s.logger.Infof(ctx, TagSystestService, "No coverage data to process for task %d", task.ID)
		return nil
	}

	// 2. 获取 Diff 信息
	var diffMap *diff.GitDiffMap
	if task.BaseCommit != "" {
		dMap, err := s.diffService.GetDiff(ctx, task.Module, task.Branch, task.Commit, task.BaseCommit)
		if err != nil {
			s.logger.Errorf(ctx, TagSystestService, "failed to get diff: %v", err)
		} else {
			diffMap = dMap
		}
	}

	// 3. 打开覆盖率报告，写入
	reportPk := partitionkey.NewSystestReportKey(task.Module, task.Branch, task.Commit)
	if err := reportPk.Unmarshal(task.ReportPartitionKey); err != nil {
		return fmt.Errorf("unmarshal report partition key failed: %w", err)
	}

	coverReport, err := s.reportManager.OpenWrite(ctx, reportPk)
	if err != nil {
		return fmt.Errorf("open report for write failed: %w", err)
	}
	defer coverReport.Close(ctx)

	// 设置元数据
	coverReport.SetMeta(report.MetaInfo{
		Module:     task.Module,
		Branch:     task.Branch,
		Commit:     task.Commit,
		BaseCommit: task.BaseCommit,
		LastUpdate: time.Now().Format(time.RFC3339),
	})

	for path, data := range coverNormalInfo {
		var fileDiffInfo report.FileDiffInfo
		if diffMap != nil {
			if dFile, ok := diffMap.DiffFileMap[path]; ok {
				fileDiffInfo.AddLines = dFile.GetAddLinesCount()
				fileDiffInfo.DeleteLines = dFile.GetDeleteLinesCount()
				for _, line := range dFile.GetFileChangeLines() {
					fileDiffInfo.AddedLines = append(fileDiffInfo.AddedLines, uint32(line))
				}
			}
		}

		if coverReport.ExistFile(path) {
			err = coverReport.UpdateFile(path, data.CoverLineData, fileDiffInfo, 0)
			if err != nil {
				return fmt.Errorf("update file %s failed: %w", path, err)
			}
		} else {
			err = coverReport.AddFile(path, data.CoverLineData, fileDiffInfo)
			if err != nil {
				return fmt.Errorf("add file %s failed: %w", path, err)
			}
		}
	}

	if err = coverReport.Flush(ctx); err != nil {
		return fmt.Errorf("flush report failed: %w", err)
	}

	// 4. 更新任务状态，并写入数据库
	task.Status = "processed"
	task.NormalCoverDataPartitionKey = req.NormalCoverDataPartitionKey
	if err := s.systestStore.Save(ctx, task); err != nil {
		s.logger.Errorf(ctx, TagSystestService, "final task save failed: %v", err)
	}

	s.logger.Infof(ctx, TagSystestService, "Task %d processed successfully", task.ID)
	return nil
}

func (s *systestService) fetchNormalInfo(ctx context.Context, task *sysStore.SystestTask) (*parser.CovNormalInfo, partitionkey.PartitionKey, error) {
	// 直接从 task 中的 NormalCoverDataPartitionKey 反序列化出 PartitionKey
	pk := partitionkey.NewEmptySystestNormalCovKey()
	if err := pk.Unmarshal(task.NormalCoverDataPartitionKey); err != nil {
		return nil, nil, fmt.Errorf("unmarshal partition key failed: %w, key: %s", err, task.NormalCoverDataPartitionKey)
	}

	jsonPath := pk.RealPathPrefix() + ".json"
	reader, err := s.ossCli.GetObject(ctx, s.bucketName, jsonPath)
	if err != nil {
		return nil, nil, fmt.Errorf("get object from oss failed: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("read data failed: %w", err)
	}

	var info parser.CovNormalInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, nil, fmt.Errorf("unmarshal json failed: %w", err)
	}
	return &info, pk, nil
}
