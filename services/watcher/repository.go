package watcher

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const HistoricalNotificationsCollectionName = "historical_notifications"
const CollectionName = "watcher"

//go:generate mockgen -source=repository.go -destination=mocks/repository_mock.go
type Repository interface {
	GetWatcher(ctx context.Context, filters bson.M) (*Watcher, error)
	GetAllWatchers(ctx context.Context) ([]*Watcher, error)
	GetWatcherList(ctx context.Context, filters bson.M, page int) ([]*Watcher, error)

	CreateWatcher(ctx context.Context, watcher *Watcher) error
	UpdateWatcher(ctx context.Context, watcher *Watcher) error
	DeleteWatcher(ctx context.Context, filters bson.M) error
	DeleteWatchersWithStaleData(ctx context.Context) error
}

type HistoryNotificationDocument struct {
	ID           primitive.ObjectID   `json:"id" bson:"_id"`
	WatcherID    primitive.ObjectID   `json:"watcher_id" bson:"watcher_id"`
	Notification *HistoryNotification `json:"notification" bson:"notification"`
}

type repository struct {
	db                                *mongo.Client
	dbName                            string
	watchersCollectionName            string
	historyNotificationCollectionName string
	logger                            *zap.SugaredLogger
}

func NewRepository(db *mongo.Client, dbName string, logger *zap.SugaredLogger) (Repository, error) {
	if db == nil {
		return nil, errors.New("[watcher_repository] invalid user database")
	}
	if dbName == "" {
		return nil, errors.New("[watcher_repository] invalid database name")
	}
	if logger == nil {
		return nil, errors.New("[watcher_repository] invalid logger")
	}

	return &repository{db: db, dbName: dbName, watchersCollectionName: CollectionName, historyNotificationCollectionName: HistoricalNotificationsCollectionName, logger: logger}, nil
}

func (r *repository) GetWatcher(ctx context.Context, filters bson.M) (*Watcher, error) {
	var watcher Watcher
	if err := r.db.Database(r.dbName).Collection(r.watchersCollectionName).FindOne(ctx, filters).Decode(&watcher); err != nil {
		if err == mongo.ErrNoDocuments {
			r.logger.Errorf("unable to find watcher: %v", err)
			// return nil, errors.New("watcher not found")
			return nil, nil
		}

		r.logger.Errorf("unable to find watcher due to internal error: %v", err)
		return nil, err
	}

	if err := r.attachHistory(ctx, &watcher); err != nil {
		r.logger.Errorf("unable to fetch and set history notifications: %v", err)
		return nil, err
	}

	return &watcher, nil
}

func (r *repository) GetAllWatchers(ctx context.Context) ([]*Watcher, error) {
	collection := r.db.Database(r.dbName).Collection(r.watchersCollectionName)

	filter := bson.D{{}}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		r.logger.Errorf("unable to find watcher due to internal error: %v", err)
		return nil, nil
	}

	defer cursor.Close(ctx)

	var watchers []*Watcher
	for cursor.Next(ctx) {
		watcher := new(Watcher)
		if err := cursor.Decode(watcher); err != nil {
			r.logger.Errorf("Unable to decode watcher document: %v", err)
			return nil, nil
		}

		if err := r.attachHistory(ctx, watcher); err != nil {
			r.logger.Errorf("unable to fetch and set history notifications: %v", err)
			return nil, nil
		}

		watchers = append(watchers, watcher)
	}

	if err := cursor.Err(); err != nil {
		r.logger.Errorf("cursor iteration error: %v", err)
		return nil, nil
	}

	return watchers, nil
}

// Fetches and assigns historical notifications to the watcher.
func (r *repository) attachHistory(ctx context.Context, watcher *Watcher) error {
	historyNotifications := make([]*HistoryNotification, 0)

	sortField := bson.E{
		Key:   "notification.timestamp",
		Value: -1,
	}
	sortCriteria := bson.D{sortField}

	cursor, err := r.db.Database(r.dbName).Collection(r.historyNotificationCollectionName).Find(ctx, bson.M{"watcher_id": watcher.ID}, options.Find().SetSort(sortCriteria))
	if err != nil {
		r.logger.Errorf("unable to find history notifications due to internal error: %v", err)
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		// fetch history notification document, set notification field in HistoryNotification struct
		historyNotificationDocument := new(HistoryNotificationDocument)
		if err := cursor.Decode(historyNotificationDocument); err != nil {
			r.logger.Errorf("Unable to decode history notification document: %v", err)
			return err
		}

		historyNotifications = append(historyNotifications, historyNotificationDocument.Notification)
	}

	if err := cursor.Err(); err != nil {
		r.logger.Errorf("cursor iteration error: %v", err)
		return err
	}

	watcher.HistoricalNotifications = &historyNotifications

	return nil
}

func (r *repository) DeleteWatchersWithStaleData(ctx context.Context) error {
	watchers, err := r.GetAllWatchers(context.Background())
	if err != nil {
		return err
	}

	for _, watcher := range watchers {
		if watcher.LastFailDate.Before(watcher.LastSuccessDate.Add(-7 * 24 * time.Hour)) {
			filter := bson.M{"_id": watcher.ID}
			if err := r.DeleteWatcher(ctx, filter); err != nil {
				return err
			}
			fmt.Printf("Watcher with ID %s deleted due to stale data\n", watcher.ID.Hex())
		}
	}

	return nil
}

func (r *repository) GetWatcherList(ctx context.Context, filters bson.M, page int) ([]*Watcher, error) {
	pageSize := 100

	skip := (page - 1) * pageSize

	findOptions := options.Find().SetSkip(int64(skip)).SetLimit(int64(pageSize))

	cur, err := r.db.Database(r.dbName).Collection(r.watchersCollectionName).Find(ctx, filters, findOptions)
	if err != nil {
		r.logger.Errorf("unable to find watcher due to internal error: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	var watchers []*Watcher
	for cur.Next(ctx) {
		watcher := new(Watcher)
		if err := cur.Decode(&watcher); err != nil {
			r.logger.Errorf("unable to decode watcher document: %v", err)
			return nil, err
		}

		if err := r.attachHistory(ctx, watcher); err != nil {
			r.logger.Errorf("unable to fetch and set history notifications: %v", err)
			return nil, err
		}
		watchers = append(watchers, watcher)
	}

	// Check for any errors that occurred during iteration
	if err := cur.Err(); err != nil {
		r.logger.Errorf("cursor iteration error: %v", err)
		return nil, err
	}

	return watchers, nil
}

func (r *repository) CreateWatcher(ctx context.Context, watcher *Watcher) error {
	// don't write historical notifications to watcher collection, set it to history notifications collection
	historicalNotifications := watcher.HistoricalNotifications
	watcher.HistoricalNotifications = nil

	_, err := r.db.Database(r.dbName).Collection(r.watchersCollectionName).InsertOne(ctx, watcher)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			r.logger.Errorf("failed to insert watcher to db due duplicate error: %s", err)
			return errors.New("watcher already exist")
		}

		r.logger.Errorf("failed to insert watcher to db: %s", err)
		return errors.New("failed to create watcher")
	}

	// add new history notifications to history notifications collection
	for _, historyNotification := range *historicalNotifications {
		historyNotificationDocument := &HistoryNotificationDocument{
			ID:           primitive.NewObjectID(),
			WatcherID:    watcher.ID,
			Notification: historyNotification,
		}
		_, err := r.db.Database(r.dbName).Collection(r.historyNotificationCollectionName).InsertOne(ctx, historyNotificationDocument)
		if err != nil {
			r.logger.Errorf("failed to insert history notification to db: %s", err)
			return errors.New("failed to create history notification")
		}
	}

	return nil
}

func (r *repository) UpdateWatcher(ctx context.Context, watcher *Watcher) error {
	historicalNotifications := watcher.HistoricalNotifications
	watcher.HistoricalNotifications = nil

	_, err := r.db.Database(r.dbName).Collection(r.watchersCollectionName).UpdateOne(ctx,
		bson.M{"_id": watcher.ID},
		bson.M{"$set": watcher})

	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			r.logger.Errorf("failed to insert watcher to db due duplicate error: %s", err)
			return errors.New("watcher already exist")
		}

		r.logger.Errorf("failed to update watcher: %s", err)
		return errors.New("failed to update watcher")
	}

	// delete all history notifications for this watcher and add historicalNotifications to history notifications collection
	_, err = r.db.Database(r.dbName).Collection(r.historyNotificationCollectionName).DeleteMany(ctx, bson.M{"watcher_id": watcher.ID})
	if err != nil {
		r.logger.Errorf("failed to delete history notifications: %v", err)
		return err
	}

	// add updated history notifications to history notifications collection
	for _, historyNotification := range *historicalNotifications {
		historyNotificationDocument := &HistoryNotificationDocument{
			ID:           primitive.NewObjectID(),
			WatcherID:    watcher.ID,
			Notification: historyNotification,
		}
		_, err := r.db.Database(r.dbName).Collection(r.historyNotificationCollectionName).InsertOne(ctx, historyNotificationDocument)
		if err != nil {
			r.logger.Errorf("failed to insert history notification to db: %s", err)
			return errors.New("failed to create history notification")
		}
	}

	return nil
}

// DeleteWatcher deletes the watcher and its associated history notifications from the database
// based on the provided filters. It retrieves the watcher using the filters and then uses its ID
// to delete both the watcher and its associated history notifications.
func (r *repository) DeleteWatcher(ctx context.Context, filters bson.M) error {
	watcher, err := r.GetWatcher(ctx, filters)
	if err != nil {
		r.logger.Errorf("unable to get watcher: %v", err)
		return err
	}

	if watcher == nil {
		r.logger.Errorf("watcher not found")
		return errors.New("watcher not found")
	}

	_, err = r.db.Database(r.dbName).Collection(r.watchersCollectionName).DeleteOne(ctx, filters)
	if err != nil {
		r.logger.Errorf("failed to delete watcher: %v", err)
		return err
	}

	_, err = r.db.Database(r.dbName).Collection(r.historyNotificationCollectionName).DeleteMany(ctx, bson.M{"watcher_id": watcher.ID})
	if err != nil {
		r.logger.Errorf("failed to delete history notifications: %v", err)
		return err
	}

	return nil
}
