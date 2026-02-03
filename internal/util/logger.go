package util

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents logging severity levels.
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

var levelNames = map[LogLevel]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

// Logger provides structured logging with levels.
type Logger struct {
	mu       sync.Mutex
	level    LogLevel
	logger   *log.Logger
	file     *os.File
	filePath string
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// GetLogger returns the default logger instance.
func GetLogger() *Logger {
	once.Do(func() {
		defaultLogger = NewLogger(LevelInfo, "")
	})
	return defaultLogger
}

// NewLogger creates a new logger with the specified level and optional file path.
func NewLogger(level LogLevel, filePath string) *Logger {
	l := &Logger{
		level:    level,
		filePath: filePath,
	}
	
	var writers []io.Writer
	writers = append(writers, os.Stdout)
	
	if filePath != "" {
		if err := EnsureDir(filepath.Dir(filePath)); err == nil {
			file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err == nil {
				l.file = file
				writers = append(writers, file)
			}
		}
	}
	
	multiWriter := io.MultiWriter(writers...)
	l.logger = log.New(multiWriter, "", 0)
	
	return l
}

// SetLevel sets the logging level.
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// ParseLevel parses a string log level.
func ParseLevel(s string) LogLevel {
	switch s {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Close closes the log file if open.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelName := levelNames[level]
	msg := fmt.Sprintf(format, args...)
	
	l.logger.Printf("[%s] %s: %s", timestamp, levelName, msg)
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an info message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Debug logs a debug message using the default logger.
func Debug(format string, args ...interface{}) {
	GetLogger().Debug(format, args...)
}

// Info logs an info message using the default logger.
func Info(format string, args ...interface{}) {
	GetLogger().Info(format, args...)
}

// Warn logs a warning message using the default logger.
func Warn(format string, args ...interface{}) {
	GetLogger().Warn(format, args...)
}

// Error logs an error message using the default logger.
func Error(format string, args ...interface{}) {
	GetLogger().Error(format, args...)
}

// InitLogger initializes the default logger with config.
func InitLogger(level string, filePath string) {
	once.Do(func() {
		defaultLogger = NewLogger(ParseLevel(level), filePath)
	})
}
