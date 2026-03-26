package service

import (
	"context"
	"testing"

	"github.com/shuaibizhang/codecoverage/idl/cover-server/coverage"
	"github.com/shuaibizhang/codecoverage/internal/diff"
	"github.com/shuaibizhang/codecoverage/internal/reports/report"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock objects
type mockReportManager struct {
	mock.Mock
}

func (m *mockReportManager) CreateReport(ctx context.Context, meta report.MetaInfo, key partitionkey.PartitionKey) (report.CoverReport, error) {
	args := m.Called(ctx, meta, key)
	return args.Get(0).(report.CoverReport), args.Error(1)
}

func (m *mockReportManager) Open(ctx context.Context, pk partitionkey.PartitionKey) (report.CoverReport, error) {
	args := m.Called(ctx, pk)
	return args.Get(0).(report.CoverReport), args.Error(1)
}

func (m *mockReportManager) OpenWrite(ctx context.Context, pk partitionkey.PartitionKey) (report.CoverReport, error) {
	args := m.Called(ctx, pk)
	return args.Get(0).(report.CoverReport), args.Error(1)
}

func (m *mockReportManager) MergeSameCommitReport(ctx context.Context, base, other report.CoverReport) error {
	args := m.Called(ctx, base, other)
	return args.Error(0)
}

func (m *mockReportManager) MergeDiffCommitReport(ctx context.Context, base, other report.CoverReport, df map[string]*diff.DiffFile) error {
	args := m.Called(ctx, base, other, df)
	return args.Error(0)
}

func (m *mockReportManager) RebaseReport(ctx context.Context, meta report.MetaInfo, rep report.CoverReport, df map[string]*diff.DiffFile) error {
	args := m.Called(ctx, meta, rep, df)
	return args.Error(0)
}

type mockCodeProvider struct {
	mock.Mock
}

func (m *mockCodeProvider) GetFileContent(ctx context.Context, repo, commit, path string) (string, error) {
	args := m.Called(ctx, repo, commit, path)
	return args.String(0), args.Error(1)
}

func (m *mockCodeProvider) ListPullRequests(ctx context.Context, repo string, state string) ([]*coverage.PullRequest, error) {
	args := m.Called(ctx, repo, state)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*coverage.PullRequest), args.Error(1)
}

func (m *mockCodeProvider) ListCommits(ctx context.Context, repo, branch string) ([]*coverage.Commit, error) {
	args := m.Called(ctx, repo, branch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*coverage.Commit), args.Error(1)
}

type mockDiffService struct {
	mock.Mock
}

func (m *mockDiffService) GetDiff(ctx context.Context, module, branch, commit, baseCommit string) (*diff.GitDiffMap, error) {
	args := m.Called(ctx, module, branch, commit, baseCommit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*diff.GitDiffMap), args.Error(1)
}

type mockCoverReport struct {
	mock.Mock
	report.CoverReport
}

func (m *mockCoverReport) GetMeta() report.MetaInfo {
	args := m.Called()
	return args.Get(0).(report.MetaInfo)
}

func (m *mockCoverReport) Flush(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockCoverReport) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestCoverageService_ListPullRequests(t *testing.T) {
	mockCP := new(mockCodeProvider)
	svc := NewCoverageService(nil, mockCP, nil, nil, nil, nil)

	ctx := context.Background()
	req := &coverage.ListPullRequestsRequest{
		Module: "owner/repo",
		State:  "open",
	}

	expectedPRs := []*coverage.PullRequest{
		{Id: 1, Title: "PR 1"},
	}

	mockCP.On("ListPullRequests", ctx, req.Module, req.State).Return(expectedPRs, nil)

	resp, err := svc.ListPullRequests(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, expectedPRs, resp.PullRequests)
	mockCP.AssertExpectations(t)
}

func TestCoverageService_ListCommits(t *testing.T) {
	mockCP := new(mockCodeProvider)
	svc := NewCoverageService(nil, mockCP, nil, nil, nil, nil)

	ctx := context.Background()
	req := &coverage.ListCommitsRequest{
		Module: "owner/repo",
		Branch: "main",
	}

	expectedCommits := []*coverage.Commit{
		{Sha: "hash1", Message: "feat: commit 1"},
	}

	mockCP.On("ListCommits", ctx, req.Module, req.Branch).Return(expectedCommits, nil)

	resp, err := svc.ListCommits(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, expectedCommits, resp.Commits)
	mockCP.AssertExpectations(t)
}

func TestCoverageService_RebaseReport(t *testing.T) {
	mockMgr := new(mockReportManager)
	mockDiff := new(mockDiffService)
	svc := NewCoverageService(mockMgr, nil, mockDiff, nil, nil, nil)

	ctx := context.Background()
	pk := partitionkey.NewReportKey(partitionkey.UnitTest, "module", "branch", "commit")
	pkStr, _ := pk.Marshal()

	req := &coverage.RebaseReportRequest{
		ReportId:   pkStr,
		BaseCommit: "new-base",
	}

	mockRep := new(mockCoverReport)
	mockMgr.On("OpenWrite", ctx, mock.Anything).Return(mockRep, nil)
	mockRep.On("GetMeta").Return(report.MetaInfo{Module: "module", Commit: "commit"})

	diffMap := &diff.GitDiffMap{
		DiffFileMap: make(map[string]*diff.DiffFile),
	}
	mockDiff.On("GetDiff", ctx, "module", "", "new-base", "commit").Return(diffMap, nil)

	mockMgr.On("RebaseReport", ctx, mock.Anything, mockRep, diffMap.DiffFileMap).Return(nil)
	mockRep.On("Flush", ctx).Return(nil)
	mockRep.On("Close", ctx).Return(nil)

	resp, err := svc.RebaseReport(ctx, req)
	assert.NoError(t, err)
	assert.True(t, resp.Success)

	mockMgr.AssertExpectations(t)
	mockDiff.AssertExpectations(t)
	mockRep.AssertExpectations(t)
}
