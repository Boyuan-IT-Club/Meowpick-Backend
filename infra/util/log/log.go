package log

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

// 全局 Logger 实例
var logger *zap.SugaredLogger

// InitLogger 初始化日志记录器。
// 必须在程序启动时（例如 main.go 的 main 函数开始时）调用一次。
func InitLogger() {
	// 使用 zapcore.EncoderConfig 配置日志的格式
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder, // 小写编码器
		EncodeTime:     zapcore.ISO8601TimeEncoder,    // ISO8601 UTC 时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder, // 短路径编码器
	}

	// 设置日志级别
	atomicLevel := zap.NewAtomicLevel()
	// TODO: 可以根据配置文件 (config.State) 来设置日志级别
	atomicLevel.SetLevel(zap.DebugLevel) // 默认为 Debug 级别

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),                   // 使用 JSON 格式
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)), // 输出到控制台
		atomicLevel, // 日志级别
	)

	// 添加调用者信息，并跳过当前文件的调用栈，以显示正确的调用位置
	l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	logger = l.Sugar()
}

// CtxInfo 记录带上下文的 Info 级别日志。
// 注意：在当前实现中，此函数行为与 Info() 相同。
// 如需记录 trace_id 等上下文信息，需要通过 Gin 中间件实现。
func CtxInfo(ctx context.Context, format string, v ...any) {
	// TODO: 从 ctx 中提取 trace_id 等信息
	// logger.With("trace_id", traceID).Infof(format, v...)
	logger.Infof(format, v...)
}

// Info 记录 Info 级别的日志
func Info(format string, v ...any) {
	logger.Infof(format, v...)
}

// CtxError 记录带上下文的 Error 级别日志。
// 注意：在当前实现中，此函数行为与 Error() 相同。
func CtxError(ctx context.Context, format string, v ...any) {
	// TODO: 从 ctx 中提取 trace_id 等信息
	logger.Errorf(format, v...)
}

// Error 记录 Error 级别的日志
func Error(format string, v ...any) {
	logger.Errorf(format, v...)
}

// CondError 当条件为 true 时记录 Error 级别的日志
func CondError(cond bool, format string, v ...any) {
	if cond {
		logger.Errorf(format, v...)
	}
}

// CtxDebug 记录带上下文的 Debug 级别日志。
// 注意：在当前实现中，此函数行为与 Debug() 相同。
func CtxDebug(ctx context.Context, format string, v ...any) {
	// TODO: 从 ctx 中提取 trace_id 等信息
	logger.Debugf(format, v...)
}

// Debug 记录 Debug 级别的日志
func Debug(format string, v ...any) {
	logger.Debugf(format, v...)
}
