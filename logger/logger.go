package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	// RedirectStdLog 重定向标准库 log 的输出到当前 logger
	RedirectStdLog() func()
}

var (
	// defaultLogger 是全局默认实例
	defaultLogger Logger
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
	setupOutput(&cfg)
	return cfg
}

// NewProductionConfig 返回一个适合生产环境的配置
func NewProductionConfig() zap.Config {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	setupOutput(&cfg)
	return cfg
}

func setupOutput(cfg *zap.Config) {
	output := os.Getenv("LOG_OUTPUT")
	logFilePath := os.Getenv("LOG_FILE_PATH")

	// 生产环境常见的策略：根据环境变量控制输出目标
	// stdout: 仅标准输出 (符合云原生/12-Factor 最佳实践)
	// file:   仅文件 (传统部署方式)
	// both:   两者都有 (过渡期或需要双备份)

	switch output {
	case "file":
		if logFilePath != "" {
			if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err == nil {
				cfg.OutputPaths = []string{logFilePath}
				cfg.ErrorOutputPaths = []string{logFilePath}
			}
		}
	case "both":
		if logFilePath != "" {
			if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err == nil {
				cfg.OutputPaths = append(cfg.OutputPaths, logFilePath)
				cfg.ErrorOutputPaths = append(cfg.ErrorOutputPaths, logFilePath)
			}
		}
	case "stdout":
		fallthrough
	default:
		// 默认只输出到 stdout
		cfg.OutputPaths = []string{"stdout"}
		cfg.ErrorOutputPaths = []string{"stderr"}
	}
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

func (l *ZapLogger) Sync() error {
	return l.z.Sync()
}

func (l *ZapLogger) RedirectStdLog() func() {
	return zap.RedirectStdLog(l.z.Desugar())
}

func (l *ZapLogger) With(keysAndValues ...interface{}) Logger {
	return &ZapLogger{z: l.z.With(keysAndValues...)}
}

// --- 包级导出函数 (Global Helper Functions) ---

func Debug(tag string, args ...interface{}) {
	Default().Debug(tag, args...)
}

// 将标准库的重定向到我们的日志
func RedirectStdLog() func() {
	return Default().RedirectStdLog()
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
