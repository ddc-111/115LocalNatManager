package service

import (
	"fmt"
	"sync"
	"time"
)

type LogLevel string

const (
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

type LogEntry struct {
	Time    string   `json:"time"`
	Level   LogLevel `json:"level"`
	Message string   `json:"message"`
}

type Logger struct {
	mu      sync.Mutex
	entries []LogEntry
	maxSize int
}

func NewLogger(maxSize int) *Logger {
	return &Logger{
		entries: make([]LogEntry, 0),
		maxSize: maxSize,
	}
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	entry := LogEntry{
		Time:    time.Now().Format("2006-01-02 15:04:05"),
		Level:   level,
		Message: msg,
	}

	l.entries = append(l.entries, entry)
	if len(l.entries) > l.maxSize {
		l.entries = l.entries[len(l.entries)-l.maxSize:]
	}
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

func (l *Logger) GetLogs(level string, limit int) []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	if limit <= 0 {
		limit = 100
	}

	var result []LogEntry
	for i := len(l.entries) - 1; i >= 0; i-- {
		if level == "" || string(l.entries[i].Level) == level {
			result = append(result, l.entries[i])
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}

func (l *Logger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = make([]LogEntry, 0)
}
