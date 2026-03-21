package processor

import (
	"context"

	"github.com/shuaibizhang/codecoverage/internal/parser"
)

type IProcessor interface {
	// GetFullCoverage 获取全量覆盖率数据
	GetFullCoverage(ctx context.Context, isAutotest bool) (*ModuleInfo, parser.CoverDataMap, error)

	// GetIncrCoverage 获取增量覆盖率数据
	GetIncrCoverage(ctx context.Context, isAutotest bool) (*ModuleInfo, parser.CoverDataMap, bool, error)

	// CleanCache 清空用于计算增量覆盖率的缓存数据
	CleanCache()

	// 检查待测服务是否就绪
	IsReady(ctx context.Context) bool
}
