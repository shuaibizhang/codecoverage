package logger

import "context"

type loggerKey struct{}

// WithContext 将 logger 注入到 context 中
func WithContext(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

// FromContext 从 context 中获取 logger，如果不存在则返回全局默认 logger
func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(loggerKey{}).(Logger); ok {
		return l
	}
	return Default()
}
