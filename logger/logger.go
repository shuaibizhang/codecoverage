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

// Logger 接口定义了日志记录器的标准行为
type Logger interface {
	// Debug 记录调试级别的日志
	Debug(tag string, args ...interface{})
	// Debugf 记录带格式的调试级别日志，并支持 context（用于提取 TraceID）
	Debugf(ctx context.Context, tag string, format string, args ...interface{})
	// Info 记录信息级别的日志
	Info(tag string, args ...interface{})
	// Infof 记录带格式的信息级别日志
	Infof(ctx context.Context, tag string, format string, args ...interface{})
	// Warn 记录警告级别的日志
	Warn(tag string, args ...interface{})
	// Warnf 记录带格式的警告级别日志
	Warnf(ctx context.Context, tag string, format string, args ...interface{})
	// Error 记录错误级别的日志
	Error(tag string, args ...interface{})
	// Errorf 记录带格式的错误级别日志
	Errorf(ctx context.Context, tag string, format string, args ...interface{})
	// With 允许创建一个带有预设字段的新 Logger
	With(keysAndValues ...interface{}) Logger
	// Sync 刷新缓冲区，确保所有日志都已写入
	Sync() error
	// RedirectStdLog 重定向标准库 log 的输出到当前 logger
	RedirectStdLog() func()
}

var (
	// defaultLogger 是全局默认实例，默认为开发配置的 ZapLogger
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

// ZapLogger 是基于 zap 的具体实现，提供了对 Logger 接口的高性能封装
type ZapLogger struct {
	z *zap.SugaredLogger
}

// NewZapLogger 使用指定的 zap 配置创建一个新的 ZapLogger 实例。
// 如果配置构建失败，将回退到 zap 的基础示例 logger。
func NewZapLogger(cfg zap.Config) *ZapLogger {
	l, err := cfg.Build(zap.AddCallerSkip(1)) // Skip 1 为了让 caller 显示正确的调用位置
	if err != nil {
		// 如果初始化失败，回退到 zap 提供的基础 Example Logger
		l = zap.NewExample()
	}
	return &ZapLogger{z: l.Sugar()}
}

// NewDevelopmentConfig 返回一个适合本地开发环境的 zap 配置。
// 该配置包含彩色日志输出，方便在终端进行调试。
func NewDevelopmentConfig() zap.Config {
	cfg := zap.NewDevelopmentConfig()
	// 使用带颜色的级别输出，方便肉眼识别
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	setupOutput(&cfg)
	return cfg
}

// NewProductionConfig 返回一个适合生产环境的 zap 配置。
// 该配置使用 JSON 格式输出，并采用 ISO8601 时间格式，方便日志系统（如 ELK）解析。
func NewProductionConfig() zap.Config {
	cfg := zap.NewProductionConfig()
	// 使用 ISO8601 时间格式，方便日志解析
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	setupOutput(&cfg)
	return cfg
}

// setupOutput 根据环境变量 LOG_OUTPUT 和 LOG_FILE_PATH 动态配置日志的输出目标。
// 支持以下输出模式：
// - stdout: 仅输出到标准输出（默认值，符合云原生 12-factor 应用原则）。
// - file:   仅输出到指定路径的日志文件。
// - both:   同时输出到标准输出和日志文件。
func setupOutput(cfg *zap.Config) {
	output := os.Getenv("LOG_OUTPUT")
	logFilePath := os.Getenv("LOG_FILE_PATH")

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

// Debug 记录一条调试级别的日志，tag 用于分类，args 为附加的键值对字段
func (l *ZapLogger) Debug(tag string, args ...interface{}) {
	l.z.Debugw(tag, args...)
}

// Debugf 记录一条带格式化的调试级别日志，并尝试从 context 中提取 TraceID 记录到日志中
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

// Info 记录一条信息级别的日志
func (l *ZapLogger) Info(tag string, args ...interface{}) {
	l.z.Infow(tag, args...)
}

// Infof 记录一条带格式化的信息级别日志，并自动关联 TraceID
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

// Warn 记录一条警告级别的日志
func (l *ZapLogger) Warn(tag string, args ...interface{}) {
	l.z.Warnw(tag, args...)
}

// Warnf 记录一条带格式化的警告级别日志，并自动关联 TraceID
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

// Error 记录一条错误级别的日志
func (l *ZapLogger) Error(tag string, args ...interface{}) {
	l.z.Errorw(tag, args...)
}

// Errorf 记录一条带格式化的错误级别日志，并自动关联 TraceID
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

// Sync 刷新日志缓冲区，确保所有日志都已写入输出目标
func (l *ZapLogger) Sync() error {
	return l.z.Sync()
}

// RedirectStdLog 将 Go 标准库 log 包的输出重定向到当前的 zap logger
func (l *ZapLogger) RedirectStdLog() func() {
	return zap.RedirectStdLog(l.z.Desugar())
}

// With 返回一个包含预设键值对的新 Logger 实例
func (l *ZapLogger) With(keysAndValues ...interface{}) Logger {
	return &ZapLogger{z: l.z.With(keysAndValues...)}
}

// --- 包级全局导出函数 (Global Helper Functions) ---

// Debug 调用全局默认 logger 的 Debug 方法
func Debug(tag string, args ...interface{}) {
	Default().Debug(tag, args...)
}

// RedirectStdLog 将标准库 log 的输出重定向到全局默认 logger
func RedirectStdLog() func() {
	return Default().RedirectStdLog()
}

// Debugf 调用全局默认 logger 的 Debugf 方法
func Debugf(ctx context.Context, tag string, format string, args ...interface{}) {
	Default().Debugf(ctx, tag, format, args...)
}

// Info 调用全局默认 logger 的 Info 方法
func Info(tag string, args ...interface{}) {
	Default().Info(tag, args...)
}

// Infof 调用全局默认 logger 的 Infof 方法
func Infof(ctx context.Context, tag string, format string, args ...interface{}) {
	Default().Infof(ctx, tag, format, args...)
}

// Warn 调用全局默认 logger 的 Warn 方法
func Warn(tag string, args ...interface{}) {
	Default().Warn(tag, args...)
}

// Warnf 调用全局默认 logger 的 Warnf 方法
func Warnf(ctx context.Context, tag string, format string, args ...interface{}) {
	Default().Warnf(ctx, tag, format, args...)
}

// Error 调用全局默认 logger 的 Error 方法
func Error(tag string, args ...interface{}) {
	Default().Error(tag, args...)
}

// Errorf 调用全局默认 logger 的 Errorf 方法
func Errorf(ctx context.Context, tag string, format string, args ...interface{}) {
	Default().Errorf(ctx, tag, format, args...)
}

// Sync 调用全局默认 logger 的 Sync 方法，通常在程序退出前调用
func Sync() error {
	return Default().Sync()
}
