package provider

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/shuaibizhang/codecoverage/internal/diff"
)

type localDiffProvider struct {
	rootDir string
}

func NewLocalDiffProvider(rootDir string) DiffProvider {
	return &localDiffProvider{rootDir: rootDir}
}

func (p *localDiffProvider) GetDiff(ctx context.Context, module, branch, commit, baseCommit string) (*diff.GitDiffMap, error) {
	// 在本地执行 git diff 命令
	// git diff baseCommit...commit
	cmd := exec.CommandContext(ctx, "git", "diff", fmt.Sprintf("%s...%s", baseCommit, commit))
	cmd.Dir = p.rootDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %v, output: %s", err, string(output))
	}

	diffStr := string(output)
	if diffStr == "" {
		return &diff.GitDiffMap{DiffFileMap: make(map[string]*diff.DiffFile)}, nil
	}

	// 解析成 GitDiffMap
	gitModel := diff.NewGitModel()
	gitDiff, err := gitModel.ParserGitDiffFile(diffStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse git diff: %w", err)
	}
	return gitDiff.CovertToMap(), nil
}
