package env

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/agent"
	"github.com/shuaibizhang/codecoverage/internal/agent/cover/openapi"
	"github.com/shuaibizhang/codecoverage/internal/agent/sut"
	"github.com/shuaibizhang/codecoverage/internal/oss"
)

type baseEnv struct {
	// 对象存储配置
	ossConfig *oss.Config
	coverApi  openapi.CoverAPI
	sutSvc    sut.ISUTService
}

func NewBaseEnv(coverApi openapi.CoverAPI, sutSvc sut.ISUTService) Environment {
	return &baseEnv{
		coverApi: coverApi,
		sutSvc:   sutSvc,
	}
}

// 已获取server下发的oss配置，已经有注册的sut，则环境准备就绪
func (e *baseEnv) IsReady(ctx context.Context) bool {
	return e.ossConfig != nil && len(e.sutSvc.GetSutMap()) > 0
}

func (e *baseEnv) Reload(ctx context.Context) error {
	// 注册agent获取下发密钥
	if e.ossConfig == nil {
		if resp, err := e.coverApi.RegistryAgentInfo(ctx, &agent.AgentRegisterRequest{}); err != nil {
			fmt.Println("RegistryAgentInfo err:", err)
			return nil
		} else {
			if resp.OssConfig != nil {
				e.ossConfig = &oss.Config{
					AccessKeyID:     resp.OssConfig.AccessKeyId,
					SecretAccessKey: resp.OssConfig.Secret,
					BucketName:      resp.OssConfig.Bucket,
					Endpoint:        resp.OssConfig.Addr,
					UseSSL:          resp.OssConfig.UseSsl,
				}
			}
		}
	}

	// 从环境变量中加载sut配置
	if !e.sutSvc.IsReady() {
		coverAddr := os.Getenv("COVER_ADDR")
		dataPath := os.Getenv("DATA_PATH")
		language := os.Getenv("LANGUAGE")
		module := os.Getenv("MODULE")
		branch := os.Getenv("BRANCH")
		commitID := os.Getenv("COMMIT_ID")
		baseCommitID := os.Getenv("BASE_COMMIT_ID")
		buildID := os.Getenv("BUILD_ID")

		if coverAddr != "" && dataPath != "" && language != "" && module != "" &&
			branch != "" && commitID != "" && baseCommitID != "" && buildID != "" {
			e.sutSvc.AddSUT(
				coverAddr,
				dataPath,
				language,
				module,
				branch,
				commitID,
				baseCommitID,
				buildID,
			)
		}
	}

	return nil
}

// 加载环境，若环境没有准备就绪，会轮询加载环境
func (e *baseEnv) Init(ctx context.Context) error {
	// 加载环境
	err := e.Reload(ctx)
	if err != nil {
		return err
	}
	if e.IsReady(ctx) {
		return nil
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for !e.IsReady(ctx) {
		select {
		case <-ctx.Done():
			err := errors.New("context canceled")
			return err
		case <-ticker.C:
			err := e.Reload(ctx)
			if err != nil {
				fmt.Println("reload env error:", err)
			}
		}
	}

	return nil
}
