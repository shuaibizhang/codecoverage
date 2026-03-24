package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/coverage"
	"github.com/shuaibizhang/codecoverage/internal/systest/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SystestController 处理系统测试相关的请求
type SystestController struct {
	svc service.SystestService
}

func NewSystestController(svc service.SystestService) *SystestController {
	return &SystestController{svc: svc}
}

// UploadSystestCoverData 上报系统测试任务元数据
func (c *SystestController) UploadSystestCoverData(ctx context.Context, req *coverage.UploadSystestCoverDataRequest) (*coverage.UploadSystestCoverDataResponse, error) {
	start := time.Now()
	log.Printf("Received UploadSystestCoverData request for module: %s, branch: %s, commit: %s", req.Module, req.Branch, req.Commit)

	id, err := c.svc.UploadSystestCoverData(ctx, req)
	log.Printf("UploadSystestCoverData service call took %v, id: %d, err: %v", time.Since(start), id, err)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save systest report: %v", err)
	}

	return &coverage.UploadSystestCoverDataResponse{
		Success: true,
		Message: fmt.Sprintf("successfully uploaded systest report with id: %d", id),
	}, nil
}
