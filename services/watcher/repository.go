package watcher

import (
	"context"
	"errors"
	"runtime"
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
	r.logger.Info("GetWatcher is called")
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
	r.logger.Info(" GetWatcher got watcher, now fetching history data")
	if err := r.attachHistory(ctx, &watcher); err != nil {
		r.logger.Errorf("unable to fetch and set history notifications: %v", err)
		return nil, err
	}
	r.logger.Info("GetWatcher got watcher history data")

	return &watcher, nil
}

func (r *repository) GetAllWatchers(ctx context.Context) ([]*Watcher, error) {
	r.logger.Info("GetAllWatchers is called")
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	r.logger.Infof("Allocated memory before getAll: %d MB", stats.Alloc/1024/1024)
	//time.Sleep(time.Second)

	collection := r.db.Database(r.dbName).Collection(r.watchersCollectionName)

	filter := bson.D{{}}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		r.logger.Errorf("unable to find watcher due to internal error: %v", err)
		return nil, nil
	}

	r.logger.Info("GetAllWatchers got all watchers")

	defer cursor.Close(ctx)

	var watchers []*Watcher
	for cursor.Next(ctx) {
		watcher := new(Watcher)
		if err := cursor.Decode(watcher); err != nil {
			r.logger.Errorf("Unable to decode watcher document: %v", err)
			return nil, nil
		}
		watchers = append(watchers, watcher)
	}

	// Check for any errors that occurred during iteration
	if err := cursor.Err(); err != nil {
		r.logger.Errorf("cursor iteration error: %v", err)
		return nil, nil
	}

	// attach notifications history to each watcher
	for _, watcher := range watchers {
		if err := r.attachHistory(ctx, watcher); err != nil {
			r.logger.Errorf("unable to fetch and set history notifications: %v", err)
			return nil, nil
		}
		r.logger.Info("GetAllWatchers got all watchers history data")

		var stats2 runtime.MemStats
		runtime.ReadMemStats(&stats2)
		r.logger.Infof("Allocated memory while attachHistory getAll: %d MB", stats2.Alloc/1024/1024)
	}

	return watchers, nil
}

// Fetches and assigns historical notifications to the watcher.
func (r *repository) attachHistory(ctx context.Context, watcher *Watcher) error {
	r.logger.Info("attachHistory is called")
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
	r.logger.Info("attachHistory got history notifications")

	if err := cursor.Err(); err != nil {
		r.logger.Errorf("cursor iteration error: %v", err)
		return err
	}

	if len(historyNotifications) > 0 {
		watcher.HistoricalNotifications = &historyNotifications
	}

	return nil
}

func (r *repository) DeleteWatchersWithStaleData(ctx context.Context) error {
	watchers, err := r.GetAllWatchers(context.Background())
	if err != nil {
		return err
	}

	r.logger.Info("got all watchers")

	for _, watcher := range watchers {
		r.logger.Info("checking watcher")
		if watcher.LastFailDate.Before(watcher.LastSuccessDate.Add(-7 * 24 * time.Hour)) {
			r.logger.Info("Delete watcher  with ID %s  \n", watcher.ID.Hex())
			filter := bson.M{"_id": watcher.ID}
			if err := r.DeleteWatcher(ctx, filter); err != nil {
				return err
			}
			r.logger.Info("Watcher with ID %s deleted due to stale data\n", watcher.ID.Hex())
		}
	}

	return nil
}

func (r *repository) GetWatcherList(ctx context.Context, filters bson.M, page int) ([]*Watcher, error) {
	r.logger.Info("GetWatcherList is called")
	pageSize := 100

	skip := (page - 1) * pageSize

	findOptions := options.Find().SetSkip(int64(skip)).SetLimit(int64(pageSize))

	cur, err := r.db.Database(r.dbName).Collection(r.watchersCollectionName).Find(ctx, filters, findOptions)
	if err != nil {
		r.logger.Errorf("unable to find watcher due to internal error: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	r.logger.Info("GetWatcherList got all watchers, now fetching history data")
	var watchers []*Watcher
	for cur.Next(ctx) {
		watcher := new(Watcher)
		if err := cur.Decode(&watcher); err != nil {
			r.logger.Errorf("unable to decode watcher document: %v", err)
			return nil, err
		}

		watchers = append(watchers, watcher)
	}

	// Check for any errors that occurred during iteration
	if err := cur.Err(); err != nil {
		r.logger.Errorf("cursor iteration error: %v", err)
		return nil, err
	}

	// attach notifications history to each watcher
	for _, watcher := range watchers {
		if err := r.attachHistory(ctx, watcher); err != nil {
			r.logger.Errorf("unable to fetch and set history notifications: %v", err)
			return nil, err
		}
		r.logger.Info("GetWatcherList got all watchers history data")
	}

	return watchers, nil
}

func (r *repository) CreateWatcher(ctx context.Context, watcher *Watcher) error {
	r.logger.Info("CreateWatcher is called")
	historicalNotifications := watcher.HistoricalNotifications

	// don't write historical notifications to watcher collection, set it to history notifications collection
	_, err := r.db.Database(r.dbName).Collection(r.watchersCollectionName).InsertOne(ctx, copyWatcherWithoutHistory(watcher))
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			r.logger.Errorf("failed to insert watcher to db due duplicate error: %s", err)
			return errors.New("watcher already exist")
		}

		r.logger.Errorf("failed to insert watcher to db: %s", err)
		return errors.New("failed to create watcher")
	}

	r.logger.Info(" CreateWatcher created watcher, now inserting history data")

	// add new history notifications to history notifications collection
	if historicalNotifications != nil {
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
	}

	r.logger.Info("CreateWatcher inserted history data")

	return nil
}

func copyWatcherWithoutHistory(watcher *Watcher) *Watcher {
	return &Watcher{
		ID:                      watcher.ID,
		PushToken:               watcher.PushToken,
		Threshold:               watcher.Threshold,
		TokenPrice:              watcher.TokenPrice,
		TxNotification:          watcher.TxNotification,
		PriceNotification:       watcher.PriceNotification,
		Addresses:               watcher.Addresses,
		LastSuccessDate:         watcher.LastSuccessDate,
		LastFailDate:            watcher.LastFailDate,
		CreatedAt:               watcher.CreatedAt,
		UpdatedAt:               watcher.UpdatedAt,
		HistoricalNotifications: nil,
	}
}

func (r *repository) UpdateWatcher(ctx context.Context, watcher *Watcher) error {
	historicalNotifications := watcher.HistoricalNotifications
	r.logger.Info("UpdateWatcher is called")

	_, err := r.db.Database(r.dbName).Collection(r.watchersCollectionName).UpdateOne(ctx,
		bson.M{"_id": watcher.ID},
		bson.M{"$set": copyWatcherWithoutHistory(watcher)})

	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			r.logger.Errorf("failed to insert watcher to db due duplicate error: %s", err)
			return errors.New("watcher already exist")
		}

		r.logger.Errorf("failed to update watcher: %s", err)
		return errors.New("failed to update watcher")
	}

	r.logger.Info("UpdateWatcher updated watcher, now updating history data")

	// delete all history notifications for this watcher and add historicalNotifications to history notifications collection
	_, err = r.db.Database(r.dbName).Collection(r.historyNotificationCollectionName).DeleteMany(ctx, bson.M{"watcher_id": watcher.ID})
	if err != nil {
		r.logger.Errorf("failed to delete history notifications: %v", err)
		return err
	}

	r.logger.Info("UpdateWatcher deleted history notifications")

	// add updated history notifications to history notifications collection
	if historicalNotifications != nil {
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
	}

	r.logger.Info("UpdateWatcher inserted history data")

	return nil
}

// DeleteWatcher deletes the watcher and its associated history notifications from the database
// based on the provided filters. It retrieves the watcher using the filters and then uses its ID
// to delete both the watcher and its associated history notifications.
func (r *repository) DeleteWatcher(ctx context.Context, filters bson.M) error {
	r.logger.Info("DeleteWatcher is called")
	watcher, err := r.GetWatcher(ctx, filters)
	if err != nil {
		r.logger.Errorf("unable to get watcher: %v", err)
		return err
	}

	r.logger.Info("DeleteWatcher got watcher to delete")

	if watcher == nil {
		r.logger.Errorf("watcher not found")
		return errors.New("watcher not found")
	}

	_, err = r.db.Database(r.dbName).Collection(r.watchersCollectionName).DeleteOne(ctx, filters)
	if err != nil {
		r.logger.Errorf("failed to delete watcher: %v", err)
		return err
	}

	r.logger.Info("DeleteWatcher deleted watcher")

	_, err = r.db.Database(r.dbName).Collection(r.historyNotificationCollectionName).DeleteMany(ctx, bson.M{"watcher_id": watcher.ID})
	if err != nil {
		r.logger.Errorf("failed to delete history notifications: %v", err)
		return err
	}

	r.logger.Info("DeleteWatcher deleted history notifications")

	return nil
}
