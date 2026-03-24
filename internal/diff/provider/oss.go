package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/shuaibizhang/codecoverage/internal/diff"
	diffStore "github.com/shuaibizhang/codecoverage/internal/diff/store"
	"github.com/shuaibizhang/codecoverage/internal/oss"
	"github.com/shuaibizhang/codecoverage/internal/reports/report/storage/partitionkey"
)

type ossDiffProvider struct {
	ossCli     oss.OSS
	diffStore  diffStore.DiffStore
	bucketName string
}

func NewOSSDiffProvider(ossCli oss.OSS, diffStore diffStore.DiffStore, bucketName string) DiffProvider {
	return &ossDiffProvider{
		ossCli:     ossCli,
		diffStore:  diffStore,
		bucketName: bucketName,
	}
}

func (p *ossDiffProvider) GetDiff(ctx context.Context, module, branch, commit, baseCommit string) (*diff.GitDiffMap, error) {
	// 1. 先查数据库是否有缓存记录
	cache, err := p.diffStore.Query(ctx, module, commit, baseCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to query diff cache from db: %w", err)
	}

	// 2. 解析 PartitionKey
	pk := partitionkey.NewDiffKey("", "", "", "")
	if err := pk.Unmarshal(cache.DiffPartitionKey); err != nil {
		return nil, fmt.Errorf("failed to unmarshal diff partition key: %w", err)
	}

	// 3. 从 OSS 获取 diff 内容
	ossPath := pk.RealPathPrefix() + ".json"
	reader, err := p.ossCli.GetObject(ctx, p.bucketName, ossPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff from oss: %w", err)
	}
	defer reader.Close()

	diffData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read diff data: %w", err)
	}

	// 4. 反序列化为 GitDiffMap
	var gitDiffMap diff.GitDiffMap
	if err := json.Unmarshal(diffData, &gitDiffMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal git diff map: %w", err)
	}

	return &gitDiffMap, nil
}
