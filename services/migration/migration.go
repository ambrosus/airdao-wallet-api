package migration

import (
	"airdao-mobile-api/services/watcher"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type HistoryNotificationDocument struct {
	WatcherID    primitive.ObjectID           `json:"watcher_id" bson:"watcher_id"`
	Notification *watcher.HistoryNotification `json:"notification" bson:"notification"`
}

type Migration struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	Name      string             `json:"name" bson:"name"`
	IsApplied bool               `json:"is_applied" bson:"is_applied"`
}

func RunMigrations(db *mongo.Client, dbName string, logger *zap.SugaredLogger) error {
	err := historicalNotificationMigration(db, dbName, logger)
	// Add more migrations here
	return err
}

func historicalNotificationMigration(db *mongo.Client, dbName string, logger *zap.SugaredLogger) error {
	if db == nil {
		return errors.New("[watcher_repository] invalid user database")
	}
	name := "historical_notification_migration"

	migrationsCollection := db.Database(dbName).Collection("migrations")
	watcherCollection := db.Database(dbName).Collection(watcher.CollectionName)
	historyCollection := db.Database(dbName).Collection(watcher.HistoricalNotificationsCollectionName)

	// Check if migration has already been run
	var migration Migration
	if err := migrationsCollection.FindOne(context.Background(), bson.M{"name": name}).Decode(&migration); err == nil {
		logger.Info("Migration already applied.")
		return nil
	}

	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "watcher_id", Value: 1},
			{Key: "timestamp", Value: -1},
		},
	}
	_, err := historyCollection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		return err
	}

	pushTokenIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "push_token", Value: 1},
		},
	}

	_, err = watcherCollection.Indexes().CreateOne(context.Background(), pushTokenIndex)
	if err != nil {
		return err
	}

	cursor, err := watcherCollection.Find(context.Background(), bson.M{})
	if err != nil {
		return err
	}
	logger.Info("Starting migration...", name)
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var w watcher.Watcher
		if err := cursor.Decode(&w); err != nil {
			return err
		}

		if w.HistoricalNotifications != nil {
			logger.Info("Migrating historical notifications...", w.ID)

			// save last 10_000 notifications
			startIndex := 0
			if len(*w.HistoricalNotifications) > 10_000 {
				startIndex = len(*w.HistoricalNotifications) - 10_000
			}
			logger.Info("len(*w.HistoricalNotifications)", len(*w.HistoricalNotifications), "startIndex", startIndex)
			for i := startIndex; i < len(*w.HistoricalNotifications); i++ {
				notification := (*w.HistoricalNotifications)[i]
				notification.ID = primitive.NewObjectID()
				notification.WatcherID = w.ID
				_, err := historyCollection.InsertOne(context.Background(), notification)
				if err != nil {
					return err
				}
			}
			// Update watcher to set historical_notifications to nil
			_, err := watcherCollection.UpdateOne(
				context.Background(),
				bson.M{"_id": w.ID},
				bson.M{"$set": bson.M{"historical_notifications": nil}},
			)
			if err != nil {
				return err
			}
		} else {
			// Log a message or handle the case when historical_notifications is nil
			logger.Info("No historical notifications found for watcher:", w.ID)
		}
	}

	if err := cursor.Err(); err != nil {
		return err
	}

	logger.Info("Migration completed successfully.")

	// Mark migration as applied
	_, err = migrationsCollection.InsertOne(context.Background(), &Migration{
		Name:      name,
		IsApplied: true,
	})
	if err != nil {
		return err
	}

	// Pause execution for index to be fully formed
	time.Sleep(5 * time.Second)

	return nil
}
