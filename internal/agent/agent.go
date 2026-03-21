package agent

import (
	"context"
	"log"
	"net/http"

	"github.com/shuaibizhang/codecoverage/internal/agent/controller"
	"github.com/shuaibizhang/codecoverage/internal/agent/env"
	"github.com/shuaibizhang/codecoverage/internal/agent/sut"

	pb "github.com/shuaibizhang/codecoverage/idl/cover-agent"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type Agent struct {
	gwmux         *runtime.ServeMux
	taskScheduler sut.ITaskSchedulerWithSutObserver // 定时任务
	sutService    sut.ISUTService
	addr          string
	env           env.Environment
}

func NewAgent(taskScheduler sut.ITaskSchedulerWithSutObserver, sutService sut.ISUTService, addr string, env env.Environment) Agent {
	return Agent{
		gwmux:         runtime.NewServeMux(),
		taskScheduler: taskScheduler,
		sutService:    sutService,
		addr:          addr,
		env:           env,
	}
}

func (a *Agent) Start(ctx context.Context) {
	// 启动服务器
	err := a.startServer(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 环境初始化，阻塞直到环境就绪（有sut待采集，下发oss密钥成功）
	err = a.env.Init(ctx)
	if err != nil {
		log.Fatal(err)
	}

	a.taskScheduler.Run(ctx)
}

func (a *Agent) startServer(ctx context.Context) error {
	// 注册grpc gateway
	err := pb.RegisterRegisterServiceHandlerServer(ctx, a.gwmux, controller.NewRegisterController(a.sutService, a.taskScheduler))
	if err != nil {
		return err
	}

	gwServer := &http.Server{
		Addr:    a.addr,
		Handler: a.gwmux,
	}

	log.Println("Serving HTTP Gateway on http://0.0.0.0:8180")
	go func() {
		gwServer.ListenAndServe()
	}()

	return nil
}
