package core

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
)

type Logger struct {
	level  LogLevel
	prefix string
	mu     sync.Mutex
}

var (
	loggers    = make(map[string]*Logger)
	loggersMu  sync.RWMutex
	rootLogger = &Logger{level: LogInfo}
)

func GetLogger(name string) *Logger {
	loggersMu.RLock()
	if l, ok := loggers[name]; ok {
		loggersMu.RUnlock()
		return l
	}
	loggersMu.RUnlock()

	loggersMu.Lock()
	defer loggersMu.Unlock()
	if l, ok := loggers[name]; ok {
		return l
	}
	l := &Logger{
		level:  rootLogger.level,
		prefix: fmt.Sprintf("[%s] ", name),
	}
	loggers[name] = l
	return l
}

func SetLogLevel(level string) {
	var l LogLevel
	switch level {
	case "debug":
		l = LogDebug
	case "info":
		l = LogInfo
	case "warn":
		l = LogWarn
	case "error":
		l = LogError
	default:
		l = LogInfo
	}
	rootLogger.level = l
	loggersMu.Lock()
	for _, logger := range loggers {
		logger.level = l
	}
	loggersMu.Unlock()
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	levelStr := ""
	switch level {
	case LogDebug:
		levelStr = "DEBUG"
	case LogInfo:
		levelStr = "INFO"
	case LogWarn:
		levelStr = "WARN"
	case LogError:
		levelStr = "ERROR"
	}

	msg := fmt.Sprintf(format, args...)
	log.Printf("%s %s%s %s", time.Now().Format("2006-01-02 15:04:05"), l.prefix, levelStr, msg)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LogDebug, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LogInfo, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LogWarn, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LogError, format, args...)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LogError, format, args...)
	os.Exit(1)
}
