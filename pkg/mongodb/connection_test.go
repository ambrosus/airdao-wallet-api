package mongodb_test

import (
	"airdao-mobile-api/config"
	"airdao-mobile-api/pkg/mongodb"
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestNewConnection(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	cfg := &config.Config{
		MongoDb: config.MongoDb{
			MongoDbName: "Test",
			MongoDbUrl:  "mongodb://localhost:27017",
		},
	}

	tests := []struct {
		name   string
		config *config.Config
		expect func(*testing.T, *mongo.Client, context.Context, context.CancelFunc, error)
	}{
		{
			name:   "should return mongo",
			config: cfg,
			expect: func(t *testing.T, client *mongo.Client, ctx context.Context, cancel context.CancelFunc, err error) {
				assert.NotNil(t, client)
				assert.NotNil(t, ctx)
				assert.NotNil(t, cancel)
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, ctx, cancel, err := mongodb.NewConnection(tc.config)
			tc.expect(t, client, ctx, cancel, err)
		})
	}
}
