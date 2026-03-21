package openapi

import (
	"context"
	"github.com/shuaibizhang/codecoverage/idl/cover-server/agent"
)

type CoverAPI interface {
	RegistryAgentInfo(ctx context.Context, req *agent.AgentRegisterRequest) (*agent.AgentRegisterResponse, error)
}

type coverAPI struct {
}

func NewCoverAPI() CoverAPI {
	return &coverAPI{}
}

func (c *coverAPI) RegistryAgentInfo(ctx context.Context, req *agent.AgentRegisterRequest) (*agent.AgentRegisterResponse, error) {
	return &agent.AgentRegisterResponse{}, nil
}
