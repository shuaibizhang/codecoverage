package code_provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/go-github/v60/github"
)

type githubCodeProvider struct {
	client *github.Client
	owner  string
}

func NewGithubCodeProvider(token, owner string) CodeProvider {
	return &githubCodeProvider{
		client: github.NewClient(nil).WithAuthToken(token),
		owner:  owner,
	}
}

func (p *githubCodeProvider) GetFileContent(ctx context.Context, repo, commit, path string) (string, error) {
	// 1. 尝试从仓库路径中解析 owner 和 repo 名称
	// repo 可能的形式: "github.com/shuaibizhang/codecoverage" 或 "shuaibizhang/transparent-context"
	repoName := repo
	ownerName := p.owner

	if strings.Contains(repo, "/") {
		parts := strings.Split(repo, "/")
		// 情况1: github.com/owner/repo
		if len(parts) >= 3 && parts[0] == "github.com" {
			ownerName = parts[1]
			repoName = parts[2]
		} else if len(parts) == 2 {
			// 情况2: owner/repo
			ownerName = parts[0]
			repoName = parts[1]
		} else {
			// 其他情况: 取最后一段作为 repo 名
			repoName = parts[len(parts)-1]
		}
	}

	// 2. 处理 commit/ref
	// - 清理末尾的冒号
	// - 如果 commit 是 "latest"，则不传 Ref，让 GitHub 默认返回默认分支的内容
	ref := strings.TrimSuffix(commit, ":")
	opts := &github.RepositoryContentGetOptions{}
	if ref != "" && ref != "latest" {
		opts.Ref = ref
	}

	// 3. 获取文件内容
	fileContent, _, resp, err := p.client.Repositories.GetContents(ctx, ownerName, repoName, path, opts)
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
