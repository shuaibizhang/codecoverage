package controller

import (
	"context"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/coverage"
	"github.com/shuaibizhang/codecoverage/idl/cover-server/register"
	"github.com/shuaibizhang/codecoverage/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CoverageController 处理覆盖率相关的请求
type CoverageController struct {
	svc service.CoverageService
}

func NewCoverageController(svc service.CoverageService) *CoverageController {
	return &CoverageController{svc: svc}
}

func (c *CoverageController) GetReportInfo(ctx context.Context, req *coverage.GetReportInfoRequest) (*coverage.GetReportInfoResponse, error) {
	return c.svc.GetReportInfo(ctx, req)
}

func (c *CoverageController) GetTreeNodes(ctx context.Context, req *coverage.GetTreeNodesRequest) (*coverage.GetTreeNodesResponse, error) {
	return c.svc.GetTreeNodes(ctx, req)
}

func (c *CoverageController) GetFileCoverage(ctx context.Context, req *coverage.GetFileCoverageRequest) (*coverage.GetFileCoverageResponse, error) {
	return c.svc.GetFileCoverage(ctx, req)
}

func (c *CoverageController) GetMetadataList(ctx context.Context, req *coverage.GetMetadataListRequest) (*coverage.GetMetadataListResponse, error) {
	return c.svc.GetMetadataList(ctx, req)
}

// Controller 是总控制器，实现了 coverage.CoverageServiceServer 接口
// 它通过组合各个子控制器来完成所有 RPC 方法的实现
type Controller struct {
	coverage.UnimplementedCoverageServiceServer
	register.UnimplementedRegisterServiceServer
	*CoverageController
	*UnitTestController
	*SystestController
	*RegisterController
}

func NewController(cov *CoverageController, ut *UnitTestController, st *SystestController, reg *RegisterController) *Controller {
	return &Controller{
		CoverageController: cov,
		UnitTestController: ut,
		SystestController:  st,
		RegisterController: reg,
	}
}

func (c *Controller) GetReportInfo(ctx context.Context, req *coverage.GetReportInfoRequest) (*coverage.GetReportInfoResponse, error) {
	return c.CoverageController.GetReportInfo(ctx, req)
}

func (c *Controller) GetTreeNodes(ctx context.Context, req *coverage.GetTreeNodesRequest) (*coverage.GetTreeNodesResponse, error) {
	return c.CoverageController.GetTreeNodes(ctx, req)
}

func (c *Controller) GetFileCoverage(ctx context.Context, req *coverage.GetFileCoverageRequest) (*coverage.GetFileCoverageResponse, error) {
	return c.CoverageController.GetFileCoverage(ctx, req)
}

func (c *Controller) GetMetadataList(ctx context.Context, req *coverage.GetMetadataListRequest) (*coverage.GetMetadataListResponse, error) {
	return c.CoverageController.GetMetadataList(ctx, req)
}

func (c *Controller) UploadUnittestReport(ctx context.Context, req *coverage.UploadUnittestReportRequest) (*coverage.UploadUnittestReportResponse, error) {
	if c.UnitTestController == nil {
		return nil, status.Errorf(codes.Unimplemented, "unittest report service is not enabled")
	}
	return c.UnitTestController.UploadUnittestReport(ctx, req)
}

func (c *Controller) UploadSystestCoverData(ctx context.Context, req *coverage.UploadSystestCoverDataRequest) (*coverage.UploadSystestCoverDataResponse, error) {
	if c.SystestController == nil {
		return nil, status.Errorf(codes.Unimplemented, "systest report service is not enabled")
	}
	return c.SystestController.UploadSystestCoverData(ctx, req)
}

func (c *Controller) AgentRegister(ctx context.Context, req *register.AgentRegisterRequest) (*register.AgentRegisterResponse, error) {
	if c.RegisterController == nil {
		return nil, status.Errorf(codes.Unimplemented, "register service is not enabled")
	}
	return c.RegisterController.AgentRegister(ctx, req)
}
