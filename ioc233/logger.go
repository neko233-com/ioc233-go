package ioc233

import (
	"fmt"
	"log/slog"
	"sync"
)

var (
	// defaultLogger 默认使用 slog.Default()，导入方可以通过 slog.SetDefault() 自定义
	globalLogger     *slog.Logger = slog.Default()
	globalLoggerLock sync.RWMutex
)

// SetLogger 设置全局日志实例
// 如果传入 nil，将使用 slog.Default()
// 导入方可以传入自定义的 slog.Logger 实例，例如：
//
//	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
//	ioc233.SetLogger(logger)
func SetLogger(logger *slog.Logger) {
	globalLoggerLock.Lock()
	defer globalLoggerLock.Unlock()
	if logger == nil {
		globalLogger = slog.Default()
	} else {
		globalLogger = logger
	}
}

// GetLogger 获取当前全局日志实例
func GetLogger() *slog.Logger {
	globalLoggerLock.RLock()
	defer globalLoggerLock.RUnlock()
	return globalLogger
}

// logDebug 内部日志函数
func logDebug(format string, args ...any) {
	globalLoggerLock.RLock()
	logger := globalLogger
	globalLoggerLock.RUnlock()
	if len(args) > 0 {
		logger.Debug(fmt.Sprintf(format, args...))
	} else {
		logger.Debug(format)
	}
}

// logInfo 内部日志函数
func logInfo(format string, args ...any) {
	globalLoggerLock.RLock()
	logger := globalLogger
	globalLoggerLock.RUnlock()
	if len(args) > 0 {
		logger.Info(fmt.Sprintf(format, args...))
	} else {
		logger.Info(format)
	}
}

// logWarn 内部日志函数
func logWarn(format string, args ...any) {
	globalLoggerLock.RLock()
	logger := globalLogger
	globalLoggerLock.RUnlock()
	if len(args) > 0 {
		logger.Warn(fmt.Sprintf(format, args...))
	} else {
		logger.Warn(format)
	}
}

// logError 内部日志函数
func logError(format string, args ...any) {
	globalLoggerLock.RLock()
	logger := globalLogger
	globalLoggerLock.RUnlock()
	if len(args) > 0 {
		logger.Error(fmt.Sprintf(format, args...))
	} else {
		logger.Error(format)
	}
}
