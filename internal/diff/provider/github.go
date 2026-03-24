package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-github/v60/github"
	"github.com/shuaibizhang/codecoverage/internal/diff"
	diffStore "github.com/shuaibizhang/codecoverage/internal/diff/store"
	githubCli "github.com/shuaibizhang/codecoverage/internal/github"
	"github.com/shuaibizhang/codecoverage/internal/oss"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/shuaibizhang/codecoverage/logger"
)

const TagGithubDiffProvider = "GithubDiffProvider"

type githubDiffProvider struct {
	client     githubCli.Client
	owner      string
	ossCli     oss.OSS
	diffStore  diffStore.DiffStore
	bucketName string
	logger     logger.Logger
}

func NewGithubDiffProvider(client githubCli.Client, owner string, ossCli oss.OSS, diffStore diffStore.DiffStore, bucketName string) DiffProvider {
	return &githubDiffProvider{
		client:     client,
		owner:      owner,
		ossCli:     ossCli,
		diffStore:  diffStore,
		bucketName: bucketName,
		logger:     logger.Default(),
	}
}

func (p *githubDiffProvider) GetDiff(ctx context.Context, module, branch, commit, baseCommit string) (*diff.GitDiffMap, error) {
	// 1. 解析 repo name
	repoName := module
	ownerName := p.owner
	if strings.Contains(module, "/") {
		parts := strings.Split(module, "/")
		if len(parts) >= 3 && parts[0] == "github.com" {
			ownerName = parts[1]
			repoName = parts[2]
		} else if len(parts) == 2 {
			ownerName = parts[0]
			repoName = parts[1]
		}
	}

	// 2. 从 GitHub 获取 diff 数据
	comparison, _, err := p.client.CompareCommits(ctx, ownerName, repoName, baseCommit, commit, &github.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to compare commits on github: %w", err)
	}

	// 获取 patch 内容并拼接成完整的 diff 字符串
	var diffBuilder strings.Builder
	for _, file := range comparison.Files {
		if file.Patch != nil {
			// 构造类似 git diff 的头部信息，以便解析器工作
			diffBuilder.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", file.GetFilename(), file.GetFilename()))
			if file.GetStatus() == "renamed" {
				diffBuilder.WriteString(fmt.Sprintf("rename from %s\n", file.GetPreviousFilename()))
				diffBuilder.WriteString(fmt.Sprintf("rename to %s\n", file.GetFilename()))
			}
			diffBuilder.WriteString(fmt.Sprintf("--- a/%s\n", file.GetPreviousFilename()))
			if file.GetPreviousFilename() == "" {
				diffBuilder.WriteString(fmt.Sprintf("--- /dev/null\n"))
			}
			diffBuilder.WriteString(fmt.Sprintf("+++ b/%s\n", file.GetFilename()))
			diffBuilder.WriteString(file.GetPatch())
			diffBuilder.WriteString("\n")
		}
	}
	diffStr := diffBuilder.String()

	// 3. 解析成 GitDiffMap
	gitModel := diff.NewGitModel()
	gitDiff, err := gitModel.ParserGitDiffFile(diffStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse git diff: %w", err)
	}
	gitDiffMap := gitDiff.CovertToMap()

	// 4. 异步写入 OSS 和数据库
	go func() {
		bgCtx := context.Background()
		// 生成 PartitionKey
		pk := partitionkey.NewDiffKey(module, branch, commit, baseCommit)
		pkStr, _ := pk.Marshal()

		// 序列化为 JSON
		diffJSON, err := json.Marshal(gitDiffMap)
		if err != nil {
			p.logger.Errorf(bgCtx, TagGithubDiffProvider, "failed to marshal diff map: %v", err)
			return
		}

		// 写入 OSS
		ossPath := pk.RealPathPrefix() + ".json"
		if err := p.ossCli.PutObject(bgCtx, p.bucketName, ossPath, bytes.NewReader(diffJSON), int64(len(diffJSON))); err != nil {
			p.logger.Errorf(bgCtx, TagGithubDiffProvider, "failed to upload diff to oss: %v", err)
			return
		}

		// 写入数据库缓存
		cache := &diffStore.DiffCache{
			Module:           module,
			CommitID:         commit,
			BaseCommitID:     baseCommit,
			DiffPartitionKey: pkStr,
		}
		if err := p.diffStore.Save(bgCtx, cache); err != nil {
			p.logger.Errorf(bgCtx, TagGithubDiffProvider, "failed to save diff to store: %v", err)
		}
	}()

	return gitDiffMap, nil
}
