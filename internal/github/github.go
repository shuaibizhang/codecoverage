package github

import (
	"context"

	"github.com/google/go-github/v60/github"
)

type Client interface {
	GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (fileContent *github.RepositoryContent, directoryContent []*github.RepositoryContent, resp *github.Response, err error)
	CompareCommits(ctx context.Context, owner, repo, base, head string, opts *github.ListOptions) (*github.CommitsComparison, *github.Response, error)
	ListPullRequests(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	ListCommits(ctx context.Context, owner, repo string, opts *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error)
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

func (c *githubClient) ListPullRequests(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	return c.client.PullRequests.List(ctx, owner, repo, opts)
}

func (c *githubClient) ListCommits(ctx context.Context, owner, repo string, opts *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error) {
	return c.client.Repositories.ListCommits(ctx, owner, repo, opts)
}
