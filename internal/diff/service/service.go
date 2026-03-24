package service

import (
	"context"

	"github.com/shuaibizhang/codecoverage/internal/diff"
	"github.com/shuaibizhang/codecoverage/internal/diff/provider"
)

type Service interface {
	GetDiff(ctx context.Context, module, branch, commit, baseCommit string) (*diff.GitDiffMap, error)
}

type diffService struct {
	provider provider.DiffProvider
}

func NewDiffService(provider provider.DiffProvider) Service {
	return &diffService{
		provider: provider,
	}
}

func (s *diffService) GetDiff(ctx context.Context, module, branch, commit, baseCommit string) (*diff.GitDiffMap, error) {
	return s.provider.GetDiff(ctx, module, branch, commit, baseCommit)
}
