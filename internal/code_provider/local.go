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
		// 1. 尝试从环境变量获取 (最高优先级)
		rootDir = os.Getenv("CODE_ROOT")
	}

	if rootDir == "" {
		// 2. 尝试从当前工作目录向上查找 go.mod (寻找项目根目录)
		curr, _ := os.Getwd()
		for curr != "/" {
			if _, err := os.Stat(filepath.Join(curr, "go.mod")); err == nil {
				rootDir = curr
				break
			}
			curr = filepath.Dir(curr)
		}
	}

	if rootDir == "" {
		// 3. 兜底使用当前工作目录
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

	return "", fmt.Errorf("file not found: %s (rootDir=%s, fullPath=%s)", path, p.rootDir, fullPath)
}
