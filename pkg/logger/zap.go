package logger

import (
	"os"
	"path/filepath"

	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
	"gopkg.in/natefinch/lumberjack.v2"
)

func newZapLogger(namespace, level string) *zap.Logger {
	globalLevel := parseLevel(level)

	// Create logs directory if it doesn't exist
	logsDir := "logs"
	var fileWriter zapcore.WriteSyncer
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		// If directory creation fails, disable file logging
		fileWriter = zapcore.AddSync(os.Stdout) // Fallback to stdout
	} else {
		// File output with rotation
		fileWriter = zapcore.AddSync(&lumberjack.Logger{
			Filename:   filepath.Join(logsDir, namespace+".log"),
			MaxSize:    10, // megabytes
			MaxBackups: 3,
			MaxAge:     7, // days
			Compress:   true,
		})
	}

	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})

	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= globalLevel && lvl < zapcore.ErrorLevel
	})

	allLevels := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= globalLevel
	})

	logStdErrorWriter := zapcore.Lock(os.Stderr)
	logStdInfoWriter := zapcore.Lock(os.Stdout)

	isTTY := term.IsTerminal(int(os.Stderr.Fd()))

	// File encoder (JSON format without colors)
	fileEncoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})

	core := zapcore.NewTee(
		// Console output (colored, using existing encoder)
		zapcore.NewCore(logging.NewEncoder(4, isTTY), logStdErrorWriter, highPriority),
		zapcore.NewCore(logging.NewEncoder(4, isTTY), logStdInfoWriter, lowPriority),
		// File output (JSON format)
		zapcore.NewCore(fileEncoder, fileWriter, allLevels),
	)

	logger := zap.New(
		core,
		zap.AddCaller(), zap.AddCallerSkip(1),
		// zap.AddStacktrace(globalLevel),
	)

	logger = logger.Named(namespace)

	zap.RedirectStdLog(logger)

	return logger
}

func parseLevel(level string) zapcore.Level {
	switch level {
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelInfo:
		return zapcore.InfoLevel
	case LevelWarn:
		return zapcore.WarnLevel
	case LevelError:
		return zapcore.ErrorLevel
	case LevelDPanic:
		return zapcore.DPanicLevel
	case LevelPanic:
		return zapcore.PanicLevel
	case LevelFatal:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}
