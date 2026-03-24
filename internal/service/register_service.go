package service

import (
	"context"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/register"
	"github.com/shuaibizhang/codecoverage/internal/config"
)

type RegisterService struct {
	register.UnimplementedRegisterServiceServer
}

func NewRegisterService() *RegisterService {
	return &RegisterService{}
}

func (s *RegisterService) AgentRegister(ctx context.Context, req *register.AgentRegisterRequest) (*register.AgentRegisterResponse, error) {
	cfg := config.GetConfig().OssConfig
	return &register.AgentRegisterResponse{
		OssConfig: &register.OssConfig{
			Endpoint:        cfg.Endpoint,
			AccessKeyId:     cfg.AccessKeyID,
			SecretAccessKey: cfg.SecretAccessKey,
			UseSsl:          cfg.UseSSL,
			BucketName:      cfg.BucketName,
		},
	}, nil
}
