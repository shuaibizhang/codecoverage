package logger

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 接口定义
type Logger interface {
	Debug(tag string, args ...interface{})
	Debugf(ctx context.Context, tag string, format string, args ...interface{})
	Info(tag string, args ...interface{})
	Infof(ctx context.Context, tag string, format string, args ...interface{})
	Warn(tag string, args ...interface{})
	Warnf(ctx context.Context, tag string, format string, args ...interface{})
	Error(tag string, args ...interface{})
	Errorf(ctx context.Context, tag string, format string, args ...interface{})
	// With 允许创建一个带有预设字段的新 Logger
	With(keysAndValues ...interface{}) Logger
	// Sync 刷新缓冲区
	Sync() error
}

var (
	// defaultLogger 是全局默认实例
	defaultLogger Logger
	// traceIDGen 全局 TraceID 生成器已被 trace.go 取代
	// once 确保默认实例只初始化一次
	once sync.Once
	// mu 保护 defaultLogger 的并发写
	mu sync.RWMutex
)

func init() {
	// 默认初始化一个开发模式的 logger
	defaultLogger = NewZapLogger(NewDevelopmentConfig())
}

// SetDefault 允许用户替换全局默认 Logger
func SetDefault(l Logger) {
	mu.Lock()
	defer mu.Unlock()
	defaultLogger = l
}

// Default 获取全局默认 Logger
func Default() Logger {
	mu.RLock()
	defer mu.RUnlock()
	return defaultLogger
}

// ZapLogger 是基于 zap 的具体实现
type ZapLogger struct {
	z *zap.SugaredLogger
}

// NewZapLogger 创建一个新的 ZapLogger
func NewZapLogger(cfg zap.Config) *ZapLogger {
	l, err := cfg.Build(zap.AddCallerSkip(1)) // Skip 1 为了让 caller 显示正确的调用位置
	if err != nil {
		// 如果初始化失败，回退到标准输出
		l = zap.NewExample()
	}
	return &ZapLogger{z: l.Sugar()}
}

// NewDevelopmentConfig 返回一个适合开发的配置
func NewDevelopmentConfig() zap.Config {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return cfg
}

// NewProductionConfig 返回一个适合生产环境的配置
func NewProductionConfig() zap.Config {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return cfg
}

func (l *ZapLogger) Debug(tag string, args ...interface{}) {
	l.z.Debugw(tag, args...)
}

func (l *ZapLogger) Debugf(ctx context.Context, tag string, format string, args ...interface{}) {
	if ctx == nil {
		ctx = context.Background()
	}
	msg := fmt.Sprintf(tag+": "+format, args...)
	traceID := GetTraceID(ctx)
	if traceID != "" {
		l.z.Debugw(msg, "trace_id", traceID)
	} else {
		l.z.Debug(msg)
	}
}

func (l *ZapLogger) Info(tag string, args ...interface{}) {
	l.z.Infow(tag, args...)
}

func (l *ZapLogger) Infof(ctx context.Context, tag string, format string, args ...interface{}) {
	if ctx == nil {
		ctx = context.Background()
	}
	msg := fmt.Sprintf(tag+": "+format, args...)
	traceID := GetTraceID(ctx)
	if traceID != "" {
		l.z.Infow(msg, "trace_id", traceID)
	} else {
		l.z.Info(msg)
	}
}

func (l *ZapLogger) Warn(tag string, args ...interface{}) {
	l.z.Warnw(tag, args...)
}

func (l *ZapLogger) Warnf(ctx context.Context, tag string, format string, args ...interface{}) {
	if ctx == nil {
		ctx = context.Background()
	}
	msg := fmt.Sprintf(tag+": "+format, args...)
	traceID := GetTraceID(ctx)
	if traceID != "" {
		l.z.Warnw(msg, "trace_id", traceID)
	} else {
		l.z.Warn(msg)
	}
}

func (l *ZapLogger) Error(tag string, args ...interface{}) {
	l.z.Errorw(tag, args...)
}

func (l *ZapLogger) Errorf(ctx context.Context, tag string, format string, args ...interface{}) {
	if ctx == nil {
		ctx = context.Background()
	}
	msg := fmt.Sprintf(tag+": "+format, args...)
	traceID := GetTraceID(ctx)
	if traceID != "" {
		l.z.Errorw(msg, "trace_id", traceID)
	} else {
		l.z.Error(msg)
	}
}

func (l *ZapLogger) With(keysAndValues ...interface{}) Logger {
	return &ZapLogger{z: l.z.With(keysAndValues...)}
}

func (l *ZapLogger) Sync() error {
	return l.z.Sync()
}

// --- 包级导出函数 (Global Helper Functions) ---

func Debug(tag string, args ...interface{}) {
	Default().Debug(tag, args...)
}

func Debugf(ctx context.Context, tag string, format string, args ...interface{}) {
	Default().Debugf(ctx, tag, format, args...)
}

func Info(tag string, args ...interface{}) {
	Default().Info(tag, args...)
}

func Infof(ctx context.Context, tag string, format string, args ...interface{}) {
	Default().Infof(ctx, tag, format, args...)
}

func Warn(tag string, args ...interface{}) {
	Default().Warn(tag, args...)
}

func Warnf(ctx context.Context, tag string, format string, args ...interface{}) {
	Default().Warnf(ctx, tag, format, args...)
}

func Error(tag string, args ...interface{}) {
	Default().Error(tag, args...)
}

func Errorf(ctx context.Context, tag string, format string, args ...interface{}) {
	Default().Errorf(ctx, tag, format, args...)
}

func Sync() error {
	return Default().Sync()
}
