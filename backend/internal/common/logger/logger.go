package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	commonerrors "github.com/AlibekovAA/dh-secure-chat/backend/internal/common/errors"
)

type Fields map[string]interface{}

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

type LoggerInterface interface {
	Debug(msg string)
	DebugSampled(prob float64, msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Critical(msg string)
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Criticalf(format string, args ...any)
	Fatal(msg string)
	Fatalf(format string, args ...any)
	WithFields(ctx context.Context, fields Fields) *Entry
}

type Logger struct {
	level       LogLevel
	out         *log.Logger
	serviceName string
	mu          sync.RWMutex
	sampler     *rand.Rand
	samplerMu   sync.Mutex
}

func New(logDir, serviceName, level string) (*Logger, error) {
	l := &Logger{
		level:       INFO,
		out:         log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile),
		serviceName: "",
		sampler:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	if err := l.initialize(logDir, serviceName, level); err != nil {
		return nil, err
	}

	return l, nil
}

func (l *Logger) initialize(logDir, serviceName, level string) error {
	if logDir == "" {
		logDir = "/var/log/dh-secure-chat"
	}

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return commonerrors.ErrInternalError.WithCause(err)
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

	l.mu.Lock()
	defer l.mu.Unlock()

	l.out = log.New(multiWriter, "", log.LstdFlags)
	l.level = parseLevel(level)
	l.serviceName = serviceName
	l.sampler = rand.New(rand.NewSource(time.Now().UnixNano()))

	return nil
}

func (l *Logger) log(level LogLevel, msg string) {
	l.logWithContext(level, nil, msg)
}

func (l *Logger) logWithContext(level LogLevel, ctx context.Context, msg string) {
	l.logWithFields(level, ctx, msg, nil)
}

func (l *Logger) logWithFields(level LogLevel, ctx context.Context, msg string, fields Fields) {
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

	var fieldParts []string

	if ctx != nil {
		type contextKey string
		const traceIDKey contextKey = "trace_id"
		if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
			fieldParts = append(fieldParts, fmt.Sprintf("trace_id=%s", traceID))
		}
	}

	if len(fields) > 0 {
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, fields[k]))
		}
	}

	if len(fieldParts) > 0 {
		prefix = fmt.Sprintf("%s [%s]", prefix, strings.Join(fieldParts, " "))
	}

	file, line := l.getCallerInfo()
	l.out.Output(0, fmt.Sprintf("%s %s:%d %s", prefix, file, line, msg))
}

func (l *Logger) Debug(msg string) { l.log(DEBUG, msg) }
func (l *Logger) DebugSampled(prob float64, msg string) {
	if l.sample(prob) {
		l.log(DEBUG, msg)
	}
}
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

func (l *Logger) WithFields(ctx context.Context, fields Fields) *Entry {
	return &Entry{
		logger: l,
		ctx:    ctx,
		fields: fields,
	}
}

type Entry struct {
	logger *Logger
	ctx    context.Context
	fields Fields
}

func (e *Entry) Debug(msg string) { e.logger.logWithFields(DEBUG, e.ctx, msg, e.fields) }
func (e *Entry) DebugSampled(prob float64, msg string) {
	if e.logger.sample(prob) {
		e.logger.logWithFields(DEBUG, e.ctx, msg, e.fields)
	}
}
func (e *Entry) Info(msg string)     { e.logger.logWithFields(INFO, e.ctx, msg, e.fields) }
func (e *Entry) Warn(msg string)     { e.logger.logWithFields(WARNING, e.ctx, msg, e.fields) }
func (e *Entry) Error(msg string)    { e.logger.logWithFields(ERROR, e.ctx, msg, e.fields) }
func (e *Entry) Critical(msg string) { e.logger.logWithFields(CRITICAL, e.ctx, msg, e.fields) }

func (e *Entry) Debugf(format string, args ...any) {
	e.logger.logWithFields(DEBUG, e.ctx, fmt.Sprintf(format, args...), e.fields)
}

func (e *Entry) Infof(format string, args ...any) {
	e.logger.logWithFields(INFO, e.ctx, fmt.Sprintf(format, args...), e.fields)
}

func (e *Entry) Warnf(format string, args ...any) {
	e.logger.logWithFields(WARNING, e.ctx, fmt.Sprintf(format, args...), e.fields)
}

func (e *Entry) Errorf(format string, args ...any) {
	e.logger.logWithFields(ERROR, e.ctx, fmt.Sprintf(format, args...), e.fields)
}

func (e *Entry) Criticalf(format string, args ...any) {
	e.logger.logWithFields(CRITICAL, e.ctx, fmt.Sprintf(format, args...), e.fields)
}

func parseLevel(value string) LogLevel {
	value = strings.TrimSpace(strings.ToUpper(value))
	switch value {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARNING", "WARN":
		return WARNING
	case "ERROR":
		return ERROR
	case "CRITICAL":
		return CRITICAL
	default:
		return INFO
	}
}

func (l *Logger) sample(prob float64) bool {
	if prob <= 0 {
		return false
	}
	if prob >= 1 {
		return true
	}
	l.samplerMu.Lock()
	defer l.samplerMu.Unlock()
	return l.sampler.Float64() < prob
}

func (l *Logger) getCallerInfo() (string, int) {
	for skip := 3; skip < 10; skip++ {
		_, file, line, ok := runtime.Caller(skip)
		if !ok {
			break
		}
		fileBase := filepath.Base(file)
		if fileBase != "logger.go" && !strings.HasSuffix(fileBase, ".s") && !strings.Contains(file, "runtime/") {
			return fileBase, line
		}
	}
	return "unknown", 0
}
