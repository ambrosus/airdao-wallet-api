package logger_test

import (
	"testing"

	"airdao-mobile-api/pkg/logger"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewLoggerConfig(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tests := []struct {
		name   string
		expect func(*testing.T, logger.Config)
	}{
		{
			name: "should return logger config",
			expect: func(t *testing.T, l logger.Config) {
				assert.NotNil(t, l)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := logger.NewLoggerConfig()
			tc.expect(t, svc)
		})
	}
}
