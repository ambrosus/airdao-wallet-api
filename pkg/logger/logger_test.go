package logger_test

import (
	"errors"
	"testing"

	"airdao-mobile-api/pkg/logger"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	appEnv := "development"

	tests := []struct {
		name   string
		appEnv string
		expect func(*testing.T, logger.Logger, error)
	}{
		{
			name:   "should return logger",
			appEnv: appEnv,
			expect: func(t *testing.T, l logger.Logger, err error) {
				assert.NotNil(t, l)
				assert.Nil(t, err)
			},
		},
		{
			name:   "should return env error",
			appEnv: "",
			expect: func(t *testing.T, l logger.Logger, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, err, errors.New("[logger] invalid app env"))
				assert.Nil(t, l)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, err := logger.NewLogger(tc.appEnv)
			tc.expect(t, svc, err)
		})
	}
}
