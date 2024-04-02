package migration

import (
	"airdao-mobile-api/services/watcher"
	"errors"

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

	// Create index { watcher_id: 1, notification.timestamp: -1 }

	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "watcher_id", Value: 1},
			{Key: "notification.timestamp", Value: -1},
		},
	}
	_, err := historyCollection.Indexes().CreateOne(context.Background(), indexModel)
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
		var watcher watcher.Watcher
		if err := cursor.Decode(&watcher); err != nil {
			return err
		}

		// Check if historical_notifications field is nil
		if watcher.HistoricalNotifications != nil {
			logger.Info("Migrating historical notifications...", watcher.ID)

			// Iterate over historical notifications only if it's not nil
			for _, notification := range *watcher.HistoricalNotifications {
				_, err := historyCollection.InsertOne(context.Background(), &HistoryNotificationDocument{
					WatcherID:    watcher.ID,
					Notification: notification,
				})
				if err != nil {
					return err
				}
			}

			// Update watcher to set historical_notifications to nil
			_, err := watcherCollection.UpdateOne(
				context.Background(),
				bson.M{"_id": watcher.ID},
				bson.M{"$set": bson.M{"historical_notifications": nil}},
			)
			if err != nil {
				return err
			}
		} else {
			// Log a message or handle the case when historical_notifications is nil
			logger.Info("No historical notifications found for watcher:", watcher.ID)
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
	return nil
}
