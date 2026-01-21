package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DebugIcon  = "🔍 "  // Magnifying glass for detailed inspection
	InfoIcon   = "✅ "  // Information symbol
	WarnIcon   = "⚠️ " // Warning symbol
	ErrorIcon  = "🔴 "  // Cross mark for errors
	DPanicIcon = "😰 "  // Anxious face for development panics
	PanicIcon  = "💥 "  // Explosion for runtime panics
	FatalIcon  = "☠️ " // Skull and crossbones for fatal errors
)

// Field ...
type Field = zapcore.Field

var (
	// Int ..
	Int = zap.Int
	// String ...
	String = zap.String
	// Error ...
	Error = zap.Error
	// Bool ...
	Bool = zap.Bool
	// Any ...
	Any = zap.Any
)

// Logger ...
type LoggerI interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	DPanic(msg string, fields ...Field)
	Panic(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
}

type loggerImpl struct {
	zap *zap.Logger
}

// NewLogger ...
func NewLogger(namespace string, level string) LoggerI {
	if level == "" {
		level = LevelInfo
	}

	logger := loggerImpl{
		zap: newZapLogger(namespace, level),
	}

	return &logger
}

func (l *loggerImpl) Debug(msg string, fields ...Field) {
	l.zap.Debug(DebugIcon+msg, fields...)
}

func (l *loggerImpl) Info(msg string, fields ...Field) {
	l.zap.Info(InfoIcon+msg, fields...)
}

func (l *loggerImpl) Warn(msg string, fields ...Field) {
	l.zap.Warn(WarnIcon+msg, fields...)
}

func (l *loggerImpl) Error(msg string, fields ...Field) {
	l.zap.Error(ErrorIcon+msg, fields...)
}

func (l *loggerImpl) DPanic(msg string, fields ...Field) {
	l.zap.DPanic(DPanicIcon+msg, fields...)
}

func (l *loggerImpl) Panic(msg string, fields ...Field) {
	l.zap.Panic(PanicIcon+msg, fields...)
}

func (l *loggerImpl) Fatal(msg string, fields ...Field) {
	l.zap.Fatal(FatalIcon+msg, fields...)
}

// GetNamed ...
func GetNamed(l LoggerI, name string) LoggerI {
	switch v := l.(type) {
	case *loggerImpl:
		v.zap = v.zap.Named(name)
		return v
	default:
		l.Info("logger.GetNamed: invalid logger type")
		return l
	}
}

// WithFields ...
func WithFields(l LoggerI, fields ...Field) LoggerI {
	switch v := l.(type) {
	case *loggerImpl:
		return &loggerImpl{
			zap: v.zap.With(fields...),
		}
	default:
		l.Info("logger.WithFields: invalid logger type")
		return l
	}
}

// Cleanup ...
func Cleanup(l LoggerI) error {
	switch v := l.(type) {
	case *loggerImpl:
		return v.zap.Sync()
	default:
		l.Info("logger.Cleanup: invalid logger type")
		return nil
	}
}
