package main

import (
	"context"
	"flag"
	"log"

	"github.com/shuaibizhang/codecoverage/internal/agent"
	"github.com/shuaibizhang/codecoverage/internal/config"
	"github.com/shuaibizhang/codecoverage/logger"
)

var confPath = flag.String("c", "conf/agent-dev.toml", "config file path")

func main() {
	flag.Parse()
	ctx := context.Background()

	if confPath == nil || *confPath == "" {
		panic("conf path err")
	}
	if err := config.Init(ctx, *confPath); err != nil {
		log.Printf("Warning: failed to init config: %v", err)
	}

	addr := config.GetConfig().AgentConfig.Addr
	if addr == "" {
		addr = "0.0.0.0:2039"
	}

	// 初始化全局 Logger (基于环境变量 LOG_OUTPUT 和 LOG_FILE_PATH)
	logger.SetDefault(logger.NewZapLogger(logger.NewProductionConfig()))
	agent := agent.NewAgent(addr)
	agent.Start(ctx)
}
