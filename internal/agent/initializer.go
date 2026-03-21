package agent

import (
	"github.com/shuaibizhang/codecoverage/internal/agent/controller"
	"github.com/shuaibizhang/codecoverage/internal/agent/cover/openapi"
	"github.com/shuaibizhang/codecoverage/internal/agent/env"
	"github.com/shuaibizhang/codecoverage/internal/agent/sut"
	"github.com/shuaibizhang/codecoverage/internal/scheduler"
)

func InitializeAgent(addr string) Agent {
	// Initialize parser
	p := ProvideParser()

	// Initialize processor
	proc := ProvideProcessor(p)

	// Initialize module info
	m := ProvideModuleInfo()

	// Initialize cover client
	coverCli := ProvideCoverCli(proc, m)

	// Initialize cron controller
	cronController := controller.NewCronController(coverCli)

	// Initialize task scheduler
	baseScheduler := scheduler.NewTaskScheduler()
	taskScheduler := sut.NewTaskSchedulerWithSutObserver(baseScheduler, cronController)

	// Initialize SUT service
	sutService := sut.NewSUTService([]sut.ISutObserver{taskScheduler})

	// Initialize openapi client
	coverApi := openapi.NewCoverAPI()

	// Initialize environment
	e := env.NewBaseEnv(coverApi, sutService)

	// Create and return agent
	return NewAgent(taskScheduler, sutService, addr, e)
}
