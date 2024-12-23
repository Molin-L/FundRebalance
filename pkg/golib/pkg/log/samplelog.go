package log

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

/**
 * TODO: 同意程序可使用不同的logger来打印
 */

type SampleLogger struct {
	zapLogger *zap.Logger
}

func NewSampleLogger(config *Config) *SampleLogger {
	core := newCore(config)
	samplerCore := zapcore.NewSampler(core, config.Tick, config.First, config.Thereafter)
	samplerLogger := zap.New(samplerCore,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zap.ErrorLevel),
	)
	return &SampleLogger{
		zapLogger: samplerLogger,
	}
}

func (logger *SampleLogger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(fields, traceInfo(ctx)...)
	logger.zapLogger.Info(msg, fields...)
}
