package controller

import (
	"context"
	"fmt"

	pb "github.com/shuaibizhang/codecoverage/idl/cover-agent"
	"github.com/shuaibizhang/codecoverage/internal/agent/sut"
	"github.com/shuaibizhang/codecoverage/internal/scheduler"
)

type RegisterController struct {
	pb.UnimplementedRegisterServiceServer
	sutService    sut.ISUTService
	taskScheduler scheduler.ITaskScheduler // 定时任务
}

func NewRegisterController(sutService sut.ISUTService, taskScheduler scheduler.ITaskScheduler) *RegisterController {
	return &RegisterController{
		sutService:    sutService,
		taskScheduler: taskScheduler,
	}
}

func (s *RegisterController) RegisterSUT(ctx context.Context, req *pb.RegisterSUTRequest) (*pb.RegisterSUTResponse, error) {
	fmt.Printf("Received RegisterSUT request: %+v\n", req)

	if req.BuildInfo == nil {
		return &pb.RegisterSUTResponse{
			Code:    400,
			Message: "BuildInfo is required",
		}, nil
	}

	// In a real application, you would save the SUT info to a database or memory store.
	// For now, we just return a success response.
	s.sutService.AddSUT(req.Address, "", req.BuildInfo.Language, req.BuildInfo.Module, req.BuildInfo.Branch, req.BuildInfo.CommitId, req.BuildInfo.BaseCommitId, req.BuildInfo.BuildId)

	return &pb.RegisterSUTResponse{
		Code:    200,
		Message: "SUT registered successfully",
	}, nil
}
