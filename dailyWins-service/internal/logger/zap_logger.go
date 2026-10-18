package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dailyWins-service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type zapLogger struct {
	logger *zap.Logger
}

var _ log.Logger = (*zapLogger)(nil)

// NewLogger builds a structured logger with console + rolling file outputs.
func NewLogger(serviceName string, cfg *conf.Logging) log.Logger {
	fileCfg := normalizeFileConfig(serviceName, cfg)
	level := parseLevel(cfg)
	encoder := newEncoder()
	writer := zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(os.Stdout),
		zapcore.AddSync(newRotateWriter(fileCfg)),
	)

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoder),
		writer,
		zap.NewAtomicLevelAt(level),
	)
	zl := zap.New(
		core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	).With(
		zap.String("service", serviceName),
	)

	return &zapLogger{logger: zl}
}

func newEncoder() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stack",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     encodeTime,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func encodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

func parseLevel(cfg *conf.Logging) zapcore.Level {
	if cfg == nil {
		return zapcore.InfoLevel
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Level)) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

type fileConfig struct {
	Path       string
	Name       string
	MaxSize    int32
	MaxBackups int32
	MaxAge     int32
	Compress   bool
}

func normalizeFileConfig(serviceName string, cfg *conf.Logging) *fileConfig {
	fc := &fileConfig{
		Path:       ".",
		Name:       serviceName + ".log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   false,
	}
	if cfg == nil || cfg.File == nil {
		return fc
	}

	if cfg.File.Path != "" {
		fc.Path = cfg.File.Path
	}
	if cfg.File.Name != "" {
		fc.Name = cfg.File.Name
	}
	if cfg.File.MaxSize > 0 {
		fc.MaxSize = cfg.File.MaxSize
	}
	if cfg.File.MaxBackups > 0 {
		fc.MaxBackups = cfg.File.MaxBackups
	}
	if cfg.File.MaxAge > 0 {
		fc.MaxAge = cfg.File.MaxAge
	}
	fc.Compress = cfg.File.Compress
	return fc
}

func newRotateWriter(fileCfg *fileConfig) *lumberjack.Logger {
	path := fileCfg.Path
	if path == "" {
		path = "."
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		path = "."
	}

	filename := filepath.Join(path, fileCfg.Name)
	return &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    int(fileCfg.MaxSize),
		MaxBackups: int(fileCfg.MaxBackups),
		MaxAge:     int(fileCfg.MaxAge),
		Compress:   fileCfg.Compress,
	}
}

func (l *zapLogger) Log(level log.Level, keyvals ...interface{}) error {
	fields := make([]zap.Field, 0, len(keyvals)/2+1)
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "_kv_unpaired", true)
	}
	for i := 0; i < len(keyvals); i += 2 {
		key := fmt.Sprint(keyvals[i])
		fields = append(fields, zap.Any(key, keyvals[i+1]))
	}

	switch level {
	case log.LevelDebug:
		l.logger.Debug("", fields...)
	case log.LevelInfo:
		l.logger.Info("", fields...)
	case log.LevelWarn:
		l.logger.Warn("", fields...)
	case log.LevelError:
		l.logger.Error("", fields...)
	case log.LevelFatal:
		l.logger.Fatal("", fields...)
	default:
		l.logger.Info("", fields...)
	}
	return nil
}
