package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/shuaibizhang/codecoverage/internal/agent"
	"github.com/shuaibizhang/codecoverage/internal/config"
)

var (
	configPath = flag.String("c", "conf/dev.toml", "config file path")
)

func main() {
	flag.Parse()

	// Handle relative path
	path := *configPath
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		// Try current working directory
		absPath := filepath.Join(wd, path)
		if _, err := os.Stat(absPath); err == nil {
			path = absPath
		} else {
			// Try looking in parent directories if default config path
			if path == "conf/dev.toml" {
				// Try ../../conf/dev.toml (common for cmd/app execution)
				altPath := filepath.Join(wd, "../../", path)
				if _, err := os.Stat(altPath); err == nil {
					path = altPath
				} else {
					// Fallback to original path if not found, let InitConfig handle the error
					path = absPath
				}
			} else {
				path = absPath
			}
		}
	}

	log.Printf("Loading config from: %s", path)
	if err := config.Init(path); err != nil {
		log.Printf("Warning: failed to init config: %v", err)
	}

	addr := config.GetConfig().AgentConfig.Addr
	if addr == "" {
		addr = "0.0.0.0:8180"
	}

	agent := agent.InitializeAgent(addr)
	agent.Start(context.Background())
}
