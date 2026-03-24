package provider

import (
	"context"

	"github.com/shuaibizhang/codecoverage/internal/diff"
)

type DiffProvider interface {
	GetDiff(ctx context.Context, module, branch, commit, base_commit string) (*diff.GitDiffMap, error)
}

// cacheDiffProvider 是一个装饰器，实现了先从 OSS 获取，获取不到则从 GitHub 获取的逻辑
type cacheDiffProvider struct {
	ossProvider    DiffProvider
	githubProvider DiffProvider
}

func NewCacheDiffProvider(ossProvider, githubProvider DiffProvider) DiffProvider {
	return &cacheDiffProvider{
		ossProvider:    ossProvider,
		githubProvider: githubProvider,
	}
}

func (p *cacheDiffProvider) GetDiff(ctx context.Context, module, branch, commit, baseCommit string) (*diff.GitDiffMap, error) {
	// 1. 尝试从 OSS 获取
	diffMap, err := p.ossProvider.GetDiff(ctx, module, branch, commit, baseCommit)
	if err == nil && diffMap != nil {
		return diffMap, nil
	}

	// 2. OSS 获取不到（或者报错），从 GitHub 获取
	return p.githubProvider.GetDiff(ctx, module, branch, commit, baseCommit)
}
