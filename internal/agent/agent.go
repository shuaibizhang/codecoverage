package agent

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/shuaibizhang/codecoverage/internal/agent/controller"
	"github.com/shuaibizhang/codecoverage/internal/agent/cover"
	"github.com/shuaibizhang/codecoverage/internal/agent/cover/openapi"
	"github.com/shuaibizhang/codecoverage/internal/agent/cover/processor"
	"github.com/shuaibizhang/codecoverage/internal/agent/env"
	"github.com/shuaibizhang/codecoverage/internal/agent/sut"
	"github.com/shuaibizhang/codecoverage/internal/config"
	"github.com/shuaibizhang/codecoverage/internal/parser"
	"github.com/shuaibizhang/codecoverage/internal/scheduler"
	"github.com/shuaibizhang/codecoverage/logger"

	pb "github.com/shuaibizhang/codecoverage/idl/cover-agent"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

const TagAgent = "_agent"

type Agent struct {
	gwmux         *runtime.ServeMux
	taskScheduler sut.ITaskSchedulerWithSutObserver // 定时任务
	sutService    sut.ISUTService
	addr          string
	env           env.Environment
	logger        logger.Logger
}

func NewAgent(addr string) Agent {
	cfg := config.GetConfig()

	// 初始化解析器，解析初始化覆盖率数据
	p := parser.NewGoCovParser("")

	// 初始化处理器，从覆盖率代理处获取覆盖率数据
	coverAddr := os.Getenv("COVER_ADDR")
	if coverAddr == "" {
		coverAddr = "127.0.0.1:7778"
	}
	proc := processor.NewGoProcessor(coverAddr, p)

	// 初始化moduleinfo
	m := &processor.ModuleInfo{}

	// 初始化覆盖率采集客户端
	coverCli := cover.NewCoverCli(proc, m)

	// 初始化定时任务cron控制器
	cronController := controller.NewCronController(coverCli, logger.Default())

	// 初始化定时任务调度器
	baseScheduler := scheduler.NewTaskScheduler()
	taskScheduler := sut.NewTaskSchedulerWithSutObserver(baseScheduler, cronController)

	// Initialize SUT service
	sutService := sut.NewSUTService([]sut.ISutObserver{taskScheduler}, logger.Default())

	// 初始化覆盖率api客户端
	coverApi := openapi.NewCoverAPI(cfg.AgentConfig.CoverServerAddr)

	// 初始化环境
	e := env.NewBaseEnv(coverApi, sutService, cronController, logger.Default())

	return Agent{
		gwmux:         runtime.NewServeMux(),
		taskScheduler: taskScheduler,
		sutService:    sutService,
		addr:          addr,
		env:           e,
		logger:        logger.Default(),
	}
}

func (a *Agent) Start(ctx context.Context) {
	// 启动服务器
	err := a.startServer(ctx)
	if err != nil {
		a.logger.Errorf(ctx, TagAgent, "start server failed: %v", err)
		return
	}

	// 环境初始化，阻塞直到环境就绪（有sut待采集，下发oss密钥成功）
	err = a.env.Init(ctx)
	if err != nil {
		a.logger.Errorf(ctx, TagAgent, "env init failed: %v", err)
		return
	}

	a.taskScheduler.Run(ctx)
}

func (a *Agent) startServer(ctx context.Context) error {
	// 注册grpc gateway
	err := pb.RegisterRegisterServiceHandlerServer(ctx, a.gwmux, controller.NewRegisterController(a.sutService, a.taskScheduler, a.logger))
	if err != nil {
		return err
	}

	gwServer := &http.Server{
		Addr:    a.addr,
		Handler: a.gwmux,
	}

	log.Printf("server load success! addr: %s\n", a.addr)
	go func() {
		gwServer.ListenAndServe()
	}()

	return nil
}
