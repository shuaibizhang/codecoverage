package logger

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

// traceIDKey 是用于在 context 中存储 TraceID 的内部键类型，使用结构体以避免冲突
type traceIDKey struct{}

// TraceIDExtractor 定义了从 context 中提取 TraceID 的函数原型。
// 开发者可以通过实现此函数并调用 SetTraceIDExtractor 来对接自定义的追踪系统。
type TraceIDExtractor func(ctx context.Context) string

var (
	// extractor 是默认的 TraceID 提取器，从特定的 traceIDKey 中读取。
	extractor TraceIDExtractor = func(ctx context.Context) string {
		if ctx == nil {
			return ""
		}
		if id, ok := ctx.Value(traceIDKey{}).(string); ok {
			return id
		}
		return ""
	}
	// extractorMu 保护 extractor 的并发访问，确保线程安全。
	extractorMu sync.RWMutex
)

// SetTraceIDExtractor 允许全局替换 TraceID 的提取逻辑。
// 例如，如果你使用了 OpenTelemetry 或 Jaeger，可以设置一个从 span context 中读取 TraceID 的提取器。
func SetTraceIDExtractor(f TraceIDExtractor) {
	extractorMu.Lock()
	defer extractorMu.Unlock()
	extractor = f
}

// GetTraceID 使用当前注册的提取器从 context 中尝试提取 TraceID。
func GetTraceID(ctx context.Context) string {
	extractorMu.RLock()
	defer extractorMu.RUnlock()
	return extractor(ctx)
}

// WithTraceID 将指定的 TraceID 注入到 context 中，并返回一个新的 context。
// 该方法配合默认的提取器使用。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// NewTraceID 生成一个随机的 32 位十六进制字符串（16 字节随机数），
// 常用于在请求开始时生成全局唯一的追踪 ID。
func NewTraceID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// 如果随机数生成失败（理论上极少发生），回退到基于当前纳秒时间戳的字符串。
		return fmt.Sprintf("%d", 0)
	}
	return hex.EncodeToString(b)
}
