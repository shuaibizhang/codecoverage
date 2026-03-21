package cover

import (
	"context"
	"fmt"

	"github.com/shuaibizhang/codecoverage/internal/agent/cover/processor"
	"github.com/shuaibizhang/codecoverage/internal/parser"
)

type ICoverCli interface {
	GetFullCoverage(ctx context.Context, isAutotest bool) (*parser.CovNormalInfo, error)
	GetIncrCoverage(ctx context.Context, isAutotest bool) (*parser.CovNormalInfo, bool, error)
}

type coverCli struct {
	processor  processor.IProcessor
	moduleInfo *processor.ModuleInfo
}

func NewCoverCli(processor processor.IProcessor, moduleInfo *processor.ModuleInfo) ICoverCli {
	return &coverCli{
		processor:  processor,
		moduleInfo: moduleInfo,
	}
}

func (f *coverCli) GetFullCoverage(ctx context.Context, isAutotest bool) (*parser.CovNormalInfo, error) {
	if f.processor == nil {
		return nil, fmt.Errorf("processor is nil")
	}

	// 获取覆盖率数据
	moduleInfo, coverData, err := f.processor.GetFullCoverage(ctx, isAutotest)
	if err != nil {
		return nil, fmt.Errorf("get full coverage data: %w", err)
	}
	// 自动化测试调用该方法时，如果获取不到moduleInfo和coverData，则直接返回
	if isAutotest && (moduleInfo == nil && coverData == nil) {
		return nil, nil
	}

	return f.handleCoverageData(ctx, moduleInfo, coverData)
}

func (f *coverCli) GetIncrCoverage(ctx context.Context, isAutotest bool) (*parser.CovNormalInfo, bool, error) {
	if f.processor == nil {
		return nil, false, fmt.Errorf("processor is nil")
	}

	// 获取覆盖率数据
	moduleInfo, coverData, isIncr, covErr := f.processor.GetIncrCoverage(ctx, isAutotest)
	if covErr != nil {
		return nil, false, fmt.Errorf("get incr coverage data: %w", covErr)
	}

	f.moduleInfo = moduleInfo

	coverInfo, err := f.handleCoverageData(ctx, moduleInfo, coverData)
	if err != nil {
		return nil, false, err
	}
	return coverInfo, isIncr, nil
}

func (f *coverCli) handleCoverageData(ctx context.Context, moduleInfo *processor.ModuleInfo,
	coverData parser.CoverDataMap) (*parser.CovNormalInfo, error) {
	if moduleInfo == nil {
		return nil, fmt.Errorf("module info is nil")
	}

	// 压缩公共前缀
	prefix, incrData := parser.CompressCommonPrefix(coverData)

	return &parser.CovNormalInfo{
		MetaInfo: parser.MetaInfo{
			Language:   string(moduleInfo.Language),
			Module:     moduleInfo.Module,
			Branch:     moduleInfo.Branch,
			Commit:     moduleInfo.Commit,
			BaseCommit: moduleInfo.BaseCommit,
			FilePrefix: prefix,
		},
		CoverageData: incrData,
	}, nil
}
