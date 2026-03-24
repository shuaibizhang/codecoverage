package controller

import (
	"context"

	pb "github.com/shuaibizhang/codecoverage/idl/cover-agent"
	"github.com/shuaibizhang/codecoverage/internal/agent/sut"
	"github.com/shuaibizhang/codecoverage/internal/scheduler"
	"github.com/shuaibizhang/codecoverage/logger"
)

const TagRegisterController = "RegisterController"

type RegisterController struct {
	pb.UnimplementedRegisterServiceServer
	sutService    sut.ISUTService
	taskScheduler scheduler.ITaskScheduler // 定时任务
	logger        logger.Logger
}

func NewRegisterController(sutService sut.ISUTService, taskScheduler scheduler.ITaskScheduler, logger logger.Logger) *RegisterController {
	return &RegisterController{
		sutService:    sutService,
		taskScheduler: taskScheduler,
		logger:        logger,
	}
}

func (s *RegisterController) RegisterSUT(ctx context.Context, req *pb.RegisterSUTRequest) (*pb.RegisterSUTResponse, error) {
	s.logger.Infof(ctx, TagRegisterController, "received register sut request: %+v", req)

	if req.BuildInfo == nil {
		return &pb.RegisterSUTResponse{
			Code:    400,
			Message: "BuildInfo is required",
		}, nil
	}

	// 添加待测单元
	s.sutService.AddSUT(ctx, req.Address, "", req.BuildInfo.Language, req.BuildInfo.Module, req.BuildInfo.Branch, req.BuildInfo.CommitId, req.BuildInfo.BaseCommitId, req.BuildInfo.BuildId)

	return &pb.RegisterSUTResponse{
		Code:    200,
		Message: "SUT registered successfully",
	}, nil
}
