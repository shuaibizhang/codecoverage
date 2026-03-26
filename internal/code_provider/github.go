package code_provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/go-github/v60/github"
	"github.com/shuaibizhang/codecoverage/idl/cover-server/coverage"
	githubCli "github.com/shuaibizhang/codecoverage/internal/github"
)

type githubCodeProvider struct {
	client githubCli.Client
	owner  string
}

func NewGithubCodeProvider(client githubCli.Client, owner string) CodeProvider {
	return &githubCodeProvider{
		client: client,
		owner:  owner,
	}
}

func (p *githubCodeProvider) GetFileContent(ctx context.Context, repo, commit, path string) (string, error) {
	// 1. 尝试从仓库路径中解析 owner 和 repo 名称
	ownerName, repoName := p.parseRepo(repo)

	// 2. 处理 commit/ref
	// - 清理末尾的冒号
	// - 如果 commit 是 "latest"，则不传 Ref，让 GitHub 默认返回默认分支的内容
	ref := strings.TrimSuffix(commit, ":")
	opts := &github.RepositoryContentGetOptions{}
	if ref != "" && ref != "latest" {
		opts.Ref = ref
	}

	// 3. 获取文件内容
	fileContent, _, resp, err := p.client.GetContents(ctx, ownerName, repoName, path, opts)
	if err != nil {
		debugRef := "default"
		if opts.Ref != "" {
			debugRef = opts.Ref
		}
		status := "unknown"
		if resp != nil {
			status = resp.Status
		}
		return "", fmt.Errorf("failed to get content from github (owner=%s, repo=%s, ref=%s, status=%s): %w", ownerName, repoName, debugRef, status, err)
	}

	if fileContent == nil {
		return "", fmt.Errorf("file not found on github: %s", path)
	}

	// 2. 如果文件太大，GetContents 会返回空内容，需要通过 DownloadURL 下载
	content, err := fileContent.GetContent()
	if err == nil && content != "" {
		return content, nil
	}

	if fileContent.GetDownloadURL() != "" {
		resp, err := http.Get(fileContent.GetDownloadURL())
		if err != nil {
			return "", fmt.Errorf("failed to download file from github: %w", err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read downloaded content: %w", err)
		}
		return string(data), nil
	}

	return "", fmt.Errorf("unable to get content for file: %s", path)
}

func (p *githubCodeProvider) ListPullRequests(ctx context.Context, repo string, state string) ([]*coverage.PullRequest, error) {
	ownerName, repoName := p.parseRepo(repo)
	opts := &github.PullRequestListOptions{
		State: state,
	}

	prs, _, err := p.client.ListPullRequests(ctx, ownerName, repoName, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests from github: %w", err)
	}

	res := make([]*coverage.PullRequest, 0, len(prs))
	for _, pr := range prs {
		res = append(res, &coverage.PullRequest{
			Id:         int32(pr.GetNumber()),
			Title:      pr.GetTitle(),
			Branch:     pr.GetHead().GetRef(),
			BaseBranch: pr.GetBase().GetRef(),
			Author:     pr.GetUser().GetLogin(),
			HtmlUrl:    pr.GetHTMLURL(),
			CreatedAt:  pr.GetCreatedAt().String(),
		})
	}
	return res, nil
}

func (p *githubCodeProvider) ListCommits(ctx context.Context, repo, branch string) ([]*coverage.Commit, error) {
	ownerName, repoName := p.parseRepo(repo)
	opts := &github.CommitsListOptions{
		SHA: branch,
	}

	commits, _, err := p.client.ListCommits(ctx, ownerName, repoName, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list commits from github: %w", err)
	}

	res := make([]*coverage.Commit, 0, len(commits))
	for _, c := range commits {
		res = append(res, &coverage.Commit{
			Sha:     c.GetSHA(),
			Message: c.GetCommit().GetMessage(),
			Author:  c.GetCommit().GetAuthor().GetName(),
			Date:    c.GetCommit().GetAuthor().GetDate().String(),
		})
	}
	return res, nil
}

func (p *githubCodeProvider) parseRepo(repo string) (string, string) {
	repoName := repo
	ownerName := p.owner

	if strings.Contains(repo, "/") {
		parts := strings.Split(repo, "/")
		if len(parts) >= 3 && parts[0] == "github.com" {
			ownerName = parts[1]
			repoName = parts[2]
		} else if len(parts) == 2 {
			ownerName = parts[0]
			repoName = parts[1]
		} else {
			repoName = parts[len(parts)-1]
		}
	}
	return ownerName, repoName
}
