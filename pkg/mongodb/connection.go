package mongodb

import (
	"context"
	"log"
	"time"

	"airdao-mobile-api/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func NewConnection(cfg *config.Config) (*mongo.Client, context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDbUrl))
	//collection := client.Database(cfg.MongoDbName)

	return client, ctx, cancel, err
}

func Ping(client *mongo.Client, ctx context.Context) error {
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return err
	}
	return nil
}

func Close(client *mongo.Client, ctx context.Context, cancel context.CancelFunc) {
	defer cancel()
	defer func(client *mongo.Client, ctx context.Context) {
		if err := client.Disconnect(ctx); err != nil {
			log.Fatal(err)
		}
	}(client, ctx)
}
