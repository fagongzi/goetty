package goetty

import (
	"go.uber.org/zap"
)

var logger = zap.NewNop()

// UseLogger use logger
func UseLogger(zapLogger *zap.Logger) {
	logger = zapLogger
}
