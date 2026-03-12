package code_provider

import (
	"context"
)

// CodeProvider 定义了获取源码内容的接口
type CodeProvider interface {
	// GetFileContent 根据仓库、Commit 和文件路径获取内容
	GetFileContent(ctx context.Context, repo, commit, path string) (string, error)
}
