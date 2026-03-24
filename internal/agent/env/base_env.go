package env

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/register"
	"github.com/shuaibizhang/codecoverage/internal/agent/controller"
	"github.com/shuaibizhang/codecoverage/internal/agent/cover/openapi"
	"github.com/shuaibizhang/codecoverage/internal/agent/sut"
	"github.com/shuaibizhang/codecoverage/internal/oss"
	"github.com/shuaibizhang/codecoverage/logger"
)

const TagEnv = "_env"

type baseEnv struct {
	// 对象存储配置
	ossConfig *oss.Config
	coverApi  openapi.CoverAPI
	sutSvc    sut.ISUTService
	cronCtrl  *controller.CronController
	logger    logger.Logger
}

func NewBaseEnv(coverApi openapi.CoverAPI, sutSvc sut.ISUTService, cronCtrl *controller.CronController, logger logger.Logger) Environment {
	cronCtrl.SetCoverAPI(coverApi)
	return &baseEnv{
		coverApi: coverApi,
		sutSvc:   sutSvc,
		cronCtrl: cronCtrl,
		logger:   logger,
	}
}

// 已获取server下发的oss配置，已经有注册的sut，则环境准备就绪
func (e *baseEnv) IsReady(ctx context.Context) bool {
	return e.ossConfig != nil && len(e.sutSvc.GetSutMap()) > 0
}

func (e *baseEnv) Reload(ctx context.Context) error {
	// 注册agent获取下发密钥
	if e.ossConfig == nil {
		if resp, err := e.coverApi.RegistryAgentInfo(ctx, &register.AgentRegisterRequest{}); err != nil {
			e.logger.Errorf(ctx, TagEnv, "register agent failed! err: %v", err)
			return nil
		} else {
			if resp.OssConfig != nil {
				e.ossConfig = &oss.Config{
					Endpoint:        resp.OssConfig.Endpoint,
					AccessKeyID:     resp.OssConfig.AccessKeyId,
					SecretAccessKey: resp.OssConfig.SecretAccessKey,
					UseSSL:          resp.OssConfig.UseSsl,
					BucketName:      resp.OssConfig.BucketName,
				}
				// 初始化 OSS 客户端并更新 CronController
				ossCli, err := oss.NewMinioOSS(*e.ossConfig)
				if err != nil {
					e.logger.Errorf(ctx, TagEnv, "new minio oss client failed! err: %v", err)
				} else {
					e.cronCtrl.SetOSSClient(ossCli, e.ossConfig.BucketName)
					e.logger.Infof(ctx, TagEnv, "minio oss client initialized and set to cron controller")
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
				ctx,
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
		return fmt.Errorf("reload env error: %w", err)
	}
	if e.IsReady(ctx) {
		e.logger.Infof(ctx, TagEnv, "environment initialized successful!")
		return nil
	}

	// 启动定时器，轮询加载环境
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// 若环境未就绪，轮询加载环境，直到环境就绪或上下文取消
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

	e.logger.Infof(ctx, TagEnv, "environment initialized successful!")
	return nil
}
