package store

import "context"

// Logger 接口，允许用户适配自己的日志库（如 zap, logrus）
type Logger interface {
	Debug(ctx context.Context, msg string, keysAndValues ...interface{})
	Info(ctx context.Context, msg string, keysAndValues ...interface{})
	Error(ctx context.Context, msg string, keysAndValues ...interface{})
}

// Metrics 接口，允许用户适配自己的监控系统（如 Prometheus, VictoriaMetrics）
type Metrics interface {
	// RecordDuration 记录耗时
	RecordDuration(ctx context.Context, name string, duration float64, labels map[string]string)
	// IncCounter 计数器加一
	IncCounter(ctx context.Context, name string, labels map[string]string)
}

// NoopLogger 默认空实现
type NoopLogger struct{}

func (n *NoopLogger) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {}
func (n *NoopLogger) Info(ctx context.Context, msg string, keysAndValues ...interface{})  {}
func (n *NoopLogger) Error(ctx context.Context, msg string, keysAndValues ...interface{}) {}

// NoopMetrics 默认空实现
type NoopMetrics struct{}

func (n *NoopMetrics) RecordDuration(ctx context.Context, name string, duration float64, labels map[string]string) {
}
func (n *NoopMetrics) IncCounter(ctx context.Context, name string, labels map[string]string) {}
