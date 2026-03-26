package code_provider

import (
	"context"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/coverage"
)

// CodeProvider 定义了获取源码内容的接口
type CodeProvider interface {
	// GetFileContent 根据仓库、Commit 和文件路径获取内容
	GetFileContent(ctx context.Context, repo, commit, path string) (string, error)

	// ListPullRequests 获取 Pull Request 列表
	ListPullRequests(ctx context.Context, repo string, state string) ([]*coverage.PullRequest, error)

	// ListCommits 获取 Commit 列表
	ListCommits(ctx context.Context, repo, branch string) ([]*coverage.Commit, error)
}
