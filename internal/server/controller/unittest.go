package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/coverage"
	"github.com/shuaibizhang/codecoverage/internal/unittest/service"
	"github.com/shuaibizhang/codecoverage/internal/unittest/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnitTestController 处理单测相关的请求
type UnitTestController struct {
	svc service.UnitTestService
}

func NewUnitTestController(svc service.UnitTestService) *UnitTestController {
	return &UnitTestController{svc: svc}
}

// UploadUnittestReport 上报单测任务元数据
func (c *UnitTestController) UploadUnittestReport(ctx context.Context, req *coverage.UploadUnittestReportRequest) (*coverage.UploadUnittestReportResponse, error) {
	start := time.Now()
	log.Printf("Received UploadUnittestReport request for module: %s, branch: %s, commit: %s", req.Module, req.Branch, req.Commit)
	log.Printf("Request body size: NormalCoverDataPartitionKey length = %d", len(req.NormalCoverDataPartitionKey))

	task := &store.UnittestTask{
		Language:                    req.Language,
		Module:                      req.Module,
		Branch:                      req.Branch,
		Commit:                      req.Commit,
		BaseCommit:                  req.BaseCommit,
		RunID:                       req.RunId,
		NormalCoverDataPartitionKey: req.NormalCoverDataPartitionKey,
		ReportPartitionKey:          req.ReportPartitionKey,
		Status:                      "completed", // 默认状态
	}

	id, err := c.svc.UploadUnittestReport(ctx, task)
	log.Printf("UploadUnittestReport service call took %v, id: %d, err: %v", time.Since(start), id, err)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save unittest report: %v", err)
	}

	return &coverage.UploadUnittestReportResponse{
		Success: true,
		Message: fmt.Sprintf("successfully uploaded unittest report with id: %d", id),
	}, nil
}
