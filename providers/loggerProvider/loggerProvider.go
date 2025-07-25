package loggerProvider

import (
	"asset/providers"
	"go.uber.org/zap"
	"log"
)

type LogProvider struct {
	logger *zap.Logger
}

func NewLogProvider() providers.ZapLoggerProvider {
	return &LogProvider{}
}

func (l *LogProvider) InitLogger() {
	var err error
	l.logger, err = zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}
	zap.ReplaceGlobals(l.logger)
}

func (l *LogProvider) SyncLogger() {
	if l.logger != nil {
		_ = l.logger.Sync()
	}
}

func (l *LogProvider) GetLogger() *zap.Logger {
	return l.logger
}
