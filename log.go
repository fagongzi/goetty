package goetty

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func adjustLogger(logger *zap.Logger, options ...zap.Option) *zap.Logger {
	if logger != nil {
		return logger
	}
	return getDefaultZapLoggerWithLevel(zap.InfoLevel, options...)
}

func getDefaultZapLoggerWithLevel(level zapcore.Level, options ...zap.Option) *zap.Logger {
	options = append(options, zap.AddStacktrace(zapcore.FatalLevel), zap.AddCaller())
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevel()
	cfg.Level.SetLevel(level)
	cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	cfg.EncoderConfig.EncodeDuration = zapcore.MillisDurationEncoder
	l, _ := cfg.Build(options...)
	return l
}
