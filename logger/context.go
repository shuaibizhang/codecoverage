package logger

import "context"

// loggerKey 是用于在 context 中存储 Logger 实例的内部键类型，采用空结构体以优化性能并避免外部冲突。
type loggerKey struct{}

// WithContext 将指定的 Logger 实例注入到 context 中，并返回一个新的 context。
// 这常用于在中间件或链路开始时，将特定配置（如带有 TraceID 的）logger 传递给下游函数。
func WithContext(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

// FromContext 从 context 中尝试提取 Logger 实例。
// 如果 context 中没有注入过 logger，则返回全局默认的 logger (Default())。
// 这种模式支持在不同链路中使用不同的日志实例，同时提供安全的兜底。
func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(loggerKey{}).(Logger); ok {
		return l
	}
	return Default()
}
