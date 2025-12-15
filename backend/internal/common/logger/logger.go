package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	CRITICAL
)

var levelNames = map[LogLevel]string{
	DEBUG:    "DEBUG",
	INFO:     "INFO",
	WARNING:  "WARNING",
	ERROR:    "ERROR",
	CRITICAL: "CRITICAL",
}

type Logger struct {
	level       LogLevel
	out         *log.Logger
	serviceName string
	mu          sync.RWMutex
}

var (
	instance *Logger
	once     sync.Once
)

func GetInstance() *Logger {
	once.Do(func() {
		instance = &Logger{
			level:       INFO,
			out:         log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile),
			serviceName: "",
		}
	})
	return instance
}

func (l *Logger) Initialize(logDir, serviceName, level string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if logDir == "" {
		logDir = "/var/log/dh-secure-chat"
	}

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile := filepath.Join(logDir, "app.log")

	fileWriter := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	multiWriter := io.MultiWriter(os.Stdout, fileWriter)
	l.out = log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)

	l.level = parseLevel(level)
	l.serviceName = serviceName

	return nil
}

func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) log(level LogLevel, msg string) {
	l.mu.RLock()
	currentLevel := l.level
	service := l.serviceName
	l.mu.RUnlock()

	if level < currentLevel {
		return
	}

	prefix := levelNames[level]
	if service != "" {
		prefix = fmt.Sprintf("[%s] [%s]", prefix, service)
	} else {
		prefix = fmt.Sprintf("[%s]", prefix)
	}

	l.out.Output(3, fmt.Sprintf("%s %s", prefix, msg))
}

func (l *Logger) Debug(msg string)    { l.log(DEBUG, msg) }
func (l *Logger) Info(msg string)     { l.log(INFO, msg) }
func (l *Logger) Warn(msg string)     { l.log(WARNING, msg) }
func (l *Logger) Error(msg string)    { l.log(ERROR, msg) }
func (l *Logger) Critical(msg string) { l.log(CRITICAL, msg) }

func (l *Logger) Debugf(format string, args ...any) {
	l.log(DEBUG, fmt.Sprintf(format, args...))
}

func (l *Logger) Infof(format string, args ...any) {
	l.log(INFO, fmt.Sprintf(format, args...))
}

func (l *Logger) Warnf(format string, args ...any) {
	l.log(WARNING, fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(format string, args ...any) {
	l.log(ERROR, fmt.Sprintf(format, args...))
}

func (l *Logger) Criticalf(format string, args ...any) {
	l.log(CRITICAL, fmt.Sprintf(format, args...))
}

func (l *Logger) Fatal(msg string) {
	l.log(CRITICAL, msg)
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, args ...any) {
	l.log(CRITICAL, fmt.Sprintf(format, args...))
	os.Exit(1)
}

func parseLevel(value string) LogLevel {
	switch value {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARNING":
		return WARNING
	case "ERROR":
		return ERROR
	case "CRITICAL":
		return CRITICAL
	default:
		return INFO
	}
}
