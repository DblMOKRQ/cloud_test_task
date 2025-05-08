package logger

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.Logger
	mu sync.Mutex // Защита от копирования
}

// New создает новый экземпляр логгера
func NewLogger(production bool) (*Logger, error) {
	var config zap.Config

	if production {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	baseLogger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{
		Logger: baseLogger,
		mu:     sync.Mutex{},
	}, nil
}

// Nop возвращает no-op логгер
func (l *Logger) Nop() *zap.Logger {
	return zap.NewNop()
}
