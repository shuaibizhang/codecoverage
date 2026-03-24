package github

import (
	"context"

	"github.com/google/go-github/v60/github"
)

type Client interface {
	GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (fileContent *github.RepositoryContent, directoryContent []*github.RepositoryContent, resp *github.Response, err error)
	CompareCommits(ctx context.Context, owner, repo, base, head string, opts *github.ListOptions) (*github.CommitsComparison, *github.Response, error)
}

type githubClient struct {
	client *github.Client
}

func NewClient(token string) Client {
	return &githubClient{
		client: github.NewClient(nil).WithAuthToken(token),
	}
}

func (c *githubClient) GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (fileContent *github.RepositoryContent, directoryContent []*github.RepositoryContent, resp *github.Response, err error) {
	return c.client.Repositories.GetContents(ctx, owner, repo, path, opts)
}

func (c *githubClient) CompareCommits(ctx context.Context, owner, repo, base, head string, opts *github.ListOptions) (*github.CommitsComparison, *github.Response, error) {
	return c.client.Repositories.CompareCommits(ctx, owner, repo, base, head, opts)
}
