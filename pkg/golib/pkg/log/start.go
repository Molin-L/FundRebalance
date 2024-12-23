//go:build !mock
// +build !mock

package log

import "go.uber.org/zap/zapcore"

func StartWithLevel(logPath string, level zapcore.Level) {
	cfg := &Config{
		FileName: logPath,
		MaxSize:  100,
		Level:    level,
	}
	StartWithConfig(cfg)
}

func EnableAsyncLog() {
	asyncLogEnabled = true
}
