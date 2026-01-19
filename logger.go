package ioc233

import (
	"fmt"
	"sync"
)

// Logger 日志接口，允许用户自定义日志实现
// 如果未设置，将使用默认的静默日志（不输出任何内容）
type Logger interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
}

// defaultLogger 默认日志实现（静默）
type defaultLogger struct{}

func (l *defaultLogger) Debug(format string, args ...any) {}
func (l *defaultLogger) Info(format string, args ...any)  {}
func (l *defaultLogger) Warn(format string, args ...any)  {}
func (l *defaultLogger) Error(format string, args ...any) {}

var (
	globalLogger     Logger = &defaultLogger{}
	globalLoggerLock sync.RWMutex
)

// SetLogger 设置全局日志实例
// 如果传入 nil，将使用默认的静默日志
func SetLogger(logger Logger) {
	globalLoggerLock.Lock()
	defer globalLoggerLock.Unlock()
	if logger == nil {
		globalLogger = &defaultLogger{}
	} else {
		globalLogger = logger
	}
}

// GetLogger 获取当前全局日志实例
func GetLogger() Logger {
	globalLoggerLock.RLock()
	defer globalLoggerLock.RUnlock()
	return globalLogger
}

// logDebug 内部日志函数
func logDebug(format string, args ...any) {
	globalLoggerLock.RLock()
	logger := globalLogger
	globalLoggerLock.RUnlock()
	logger.Debug(format, args...)
}

// logInfo 内部日志函数
func logInfo(format string, args ...any) {
	globalLoggerLock.RLock()
	logger := globalLogger
	globalLoggerLock.RUnlock()
	logger.Info(format, args...)
}

// logWarn 内部日志函数
func logWarn(format string, args ...any) {
	globalLoggerLock.RLock()
	logger := globalLogger
	globalLoggerLock.RUnlock()
	logger.Warn(format, args...)
}

// logError 内部日志函数
func logError(format string, args ...any) {
	globalLoggerLock.RLock()
	logger := globalLogger
	globalLoggerLock.RUnlock()
	logger.Error(format, args...)
}

// StdLogger 标准输出日志实现（用于调试）
type StdLogger struct{}

func (l *StdLogger) Debug(format string, args ...any) {
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}

func (l *StdLogger) Info(format string, args ...any) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

func (l *StdLogger) Warn(format string, args ...any) {
	fmt.Printf("[WARN] "+format+"\n", args...)
}

func (l *StdLogger) Error(format string, args ...any) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}
