package agent

import (
	"os"

	"github.com/shuaibizhang/codecoverage/internal/agent/controller"
	"github.com/shuaibizhang/codecoverage/internal/agent/cover"
	"github.com/shuaibizhang/codecoverage/internal/agent/cover/processor"
	"github.com/shuaibizhang/codecoverage/internal/agent/sut"
	"github.com/shuaibizhang/codecoverage/internal/parser"
)

func ProvideParser() parser.Parser {
	return parser.NewGoCovParser("")
}

func ProvideModuleInfo() *processor.ModuleInfo {
	return &processor.ModuleInfo{}
}

func ProvideProcessor(p parser.Parser) processor.IProcessor {
	addr := os.Getenv("COVER_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	return processor.NewGoProcessor(addr, p)
}

func ProvideCoverCli(p processor.IProcessor, m *processor.ModuleInfo) cover.ICoverCli {
	return cover.NewCoverCli(p, m)
}

func ProvideCronController(c *controller.CronController) sut.ICronController {
	return c
}
