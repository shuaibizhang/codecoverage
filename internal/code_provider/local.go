package code_provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type localCodeProvider struct {
	rootDir string
}

func NewLocalCodeProvider(rootDir string) CodeProvider {
	if rootDir == "" {
		rootDir, _ = os.Getwd()
	}
	return &localCodeProvider{rootDir: rootDir}
}

func (p *localCodeProvider) GetFileContent(ctx context.Context, repo, commit, path string) (string, error) {
	// 1. 尝试直接作为绝对路径读取
	absPath := path
	if !filepath.IsAbs(absPath) {
		absPath = "/" + path
	}

	if _, err := os.Stat(absPath); err == nil {
		data, err := os.ReadFile(absPath)
		if err == nil {
			return string(data), nil
		}
	}

	// 2. 尝试相对于项目根目录读取
	fullPath := filepath.Join(p.rootDir, path)
	data, err := os.ReadFile(fullPath)
	if err == nil {
		return string(data), nil
	}

	return "", fmt.Errorf("file not found: %s", path)
}
