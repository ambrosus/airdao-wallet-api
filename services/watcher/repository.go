package watcher

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const HistoricalNotificationsCollectionName = "historical_notifications"
const CollectionName = "watcher"

//go:generate mockgen -source=repository.go -destination=mocks/repository_mock.go
type Repository interface {
	GetWatcher(ctx context.Context, filters bson.M) (*Watcher, error)
	GetWatcherList(ctx context.Context, filters bson.M, page int) ([]*Watcher, error)

	CreateWatcher(ctx context.Context, watcher *Watcher) error
	UpdateWatcher(ctx context.Context, watcher *Watcher) error
	DeleteWatcher(ctx context.Context, filters bson.M) error
	DeleteWatchersWithStaleData(ctx context.Context) error
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

	collection := r.db.Database(r.dbName).Collection(r.watchersCollectionName)

	var watcher Watcher
	err := collection.FindOne(ctx, filters).Decode(&watcher)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			r.logger.Errorf("watcher not found")
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

func (r *repository) getAllWatchersWithoutHistory(ctx context.Context) ([]*Watcher, error) {
	r.logger.Info("GetAllWatchersWithoutHistory is called")

	collection := r.db.Database(r.dbName).Collection(r.watchersCollectionName)

	cur, err := collection.Find(ctx, bson.M{})
	if err != nil {
		r.logger.Errorf("unable to find watchers due to internal error: %v", err)
		return nil, err
	}

	defer cur.Close(ctx)

	var watchers []*Watcher

	if err := cur.All(ctx, &watchers); err != nil {
		r.logger.Errorf("unable to decode watchers: %v", err)
		return nil, err
	}

	r.logger.Info("GetAllWatchersWithoutHistory got watchers")

	return watchers, nil
}

func (r *repository) DeleteWatchersWithStaleData(ctx context.Context) error {
	r.logger.Info("DeleteWatchersWithStaleData is called")
	watchers, err := r.getAllWatchersWithoutHistory(ctx)
	if err != nil {
		r.logger.Errorf("unable to get all watchers: %v", err)
	}
	if watchers == nil {
		r.logger.Info("no watchers found")
		return nil
	}

	for _, watcher := range watchers {
		r.logger.Info("checking watcher")
		if watcher.LastFailDate.Before(watcher.LastSuccessDate.Add(-7 * 24 * time.Hour)) {
			r.logger.Infof("Delete watcher  with ID %s  \n", watcher.ID.Hex())
			filter := bson.M{"_id": watcher.ID}
			if err := r.DeleteWatcher(ctx, filter); err != nil {
				return err
			}
			r.logger.Infof("Watcher with ID %s deleted due to stale data\n", watcher.ID.Hex())
		}
	}

	return nil
}

func (r *repository) GetWatcherList(ctx context.Context, filters bson.M, page int) ([]*Watcher, error) {
	r.logger.Info("GetPaginatedWatchers is called")

	pageSize := 100

	skip := (page - 1) * pageSize

	findOptions := options.Find().SetSkip(int64(skip)).SetLimit(int64(pageSize))

	cur, err := r.db.Database(r.dbName).Collection(r.watchersCollectionName).Find(ctx, filters, findOptions)
	if err != nil {
		r.logger.Errorf("unable to find watchers due to internal error: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	var watchers []*Watcher
	if err := cur.All(ctx, &watchers); err != nil {
		r.logger.Errorf("unable to decode watchers: %v", err)
		return nil, err
	}

	for _, watcher := range watchers {
		if err := r.attachHistory(ctx, watcher); err != nil {
			r.logger.Errorf("unable to fetch and set history notifications: %v", err)
			return nil, err
		}
		r.logger.Info("GetWatcherList got watcher history data")
	}

	return watchers, nil
}

func (r *repository) attachHistory(ctx context.Context, watcher *Watcher) error {
	r.logger.Info("attachHistory is called")
	sortField := bson.E{
		Key:   "timestamp",
		Value: -1,
	}
	sortCriteria := bson.D{sortField}
	opts := options.Find().SetSort(sortCriteria).SetLimit(10_000)
	coll := r.db.Database(r.dbName).Collection(r.historyNotificationCollectionName)

	cursor, err := coll.Find(ctx, bson.M{"watcher_id": watcher.ID}, opts)
	if err != nil {
		r.logger.Errorf("unable to find history notifications due to internal error: %v", err)
		return err
	}
	defer cursor.Close(ctx)
	var historyNotifications []*HistoryNotification
	if err := cursor.All(ctx, &historyNotifications); err != nil {
		r.logger.Errorf("unable to decode watchers: %v", err)
		return err
	}
	watcher.HistoricalNotifications = &historyNotifications

	return nil
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
			historyNotification.WatcherID = watcher.ID
			_, err := r.db.Database(r.dbName).Collection(r.historyNotificationCollectionName).InsertOne(ctx, historyNotification)
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

	if historicalNotifications == nil || len(*historicalNotifications) == 0 {
		return nil
	}
	docs := make([]any, len(*historicalNotifications))
	for i, v := range *historicalNotifications {
		docs[i] = v
	}

	_, err = r.db.Database(r.dbName).Collection(r.historyNotificationCollectionName).InsertMany(ctx, docs)
	if err != nil {
		r.logger.Errorf("failed to insert history notifications: %v", err)
		return err
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
