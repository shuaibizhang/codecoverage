package controller

import (
	"context"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/register"
	"github.com/shuaibizhang/codecoverage/internal/service"
)

type RegisterController struct {
	register.UnimplementedRegisterServiceServer
	svc *service.RegisterService
}

func NewRegisterController(svc *service.RegisterService) *RegisterController {
	return &RegisterController{
		svc: svc,
	}
}

func (c *RegisterController) AgentRegister(ctx context.Context, req *register.AgentRegisterRequest) (*register.AgentRegisterResponse, error) {
	return c.svc.AgentRegister(ctx, req)
}
