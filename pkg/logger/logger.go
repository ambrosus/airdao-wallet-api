package logger

import (
	"errors"

	"go.uber.org/zap"
)

//go:generate mockgen -source=logger.go -destination=mocks/logger_mock.go
type Logger interface {
	SetupZapLogger() (*zap.SugaredLogger, error)
}

type logger struct {
	appEnv string
}

func NewLogger(appEnv string) (Logger, error) {
	if appEnv == "" {
		return nil, errors.New("[logger] invalid app env")
	}

	return &logger{appEnv: appEnv}, nil
}

func (l *logger) SetupZapLogger() (*zap.SugaredLogger, error) {
	loggerConfig := NewLoggerConfig()

	switch l.appEnv {
	case "production":
		logger, err := loggerConfig.GetProductionConfig().Config.Build()
		if err != nil {
			return nil, err
		}
		return logger.Sugar(), nil
	case "development":
		logger, err := loggerConfig.GetDevelopmentConfig().Config.Build()
		if err != nil {
			return nil, err
		}
		return logger.Sugar(), nil
	}

	return nil, errors.New("[logger] incorrect app env")
}
