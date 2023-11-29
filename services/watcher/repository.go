package watcher

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

//go:generate mockgen -source=repository.go -destination=mocks/repository_mock.go
type Repository interface {
	GetWatcher(ctx context.Context, filters bson.M) (*Watcher, error)
	GetWatcherList(ctx context.Context, filters bson.M, page int) ([]*Watcher, error)

	CreateWatcher(ctx context.Context, watcher *Watcher) error
	UpdateWatcher(ctx context.Context, watcher *Watcher) error
	DeleteWatcher(ctx context.Context, filters bson.M) error
}

type repository struct {
	db               *mongo.Client
	dbName           string
	dbCollectionName string
	logger           *zap.SugaredLogger
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

	return &repository{db: db, dbName: dbName, dbCollectionName: "watcher", logger: logger}, nil
}

func (r *repository) GetWatcher(ctx context.Context, filters bson.M) (*Watcher, error) {
	var watcher Watcher

	if err := r.db.Database(r.dbName).Collection(r.dbCollectionName).FindOne(ctx, filters).Decode(&watcher); err != nil {
		if err == mongo.ErrNoDocuments {
			r.logger.Errorf("unable to find watcher: %v", err)
			// return nil, errors.New("watcher not found")
			return nil, nil
		}

		r.logger.Errorf("unable to find watcher due to internal error: %v", err)
		return nil, err
	}

	return &watcher, nil
}

func (r *repository) GetWatcherList(ctx context.Context, filters bson.M, page int) ([]*Watcher, error) {
	pageSize := 100

	skip := (page - 1) * pageSize

	findOptions := options.Find().SetSkip(int64(skip)).SetLimit(int64(pageSize))

	cur, err := r.db.Database(r.dbName).Collection(r.dbCollectionName).Find(ctx, filters, findOptions)
	if err != nil {
		r.logger.Errorf("unable to find watcher due to internal error: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	var watchers []*Watcher
	for cur.Next(ctx) {
		var watcher Watcher
		if err := cur.Decode(&watcher); err != nil {
			r.logger.Errorf("unable to decode watcher document: %v", err)
			return nil, err
		}
		watchers = append(watchers, &watcher)
	}


	// Check for any errors that occurred during iteration
	if err := cur.Err(); err != nil {
		r.logger.Errorf("cursor iteration error: %v", err)
		return nil, err
	}

	return watchers, nil
}

func (r *repository) CreateWatcher(ctx context.Context, watcher *Watcher) error {
	_, err := r.db.Database(r.dbName).Collection(r.dbCollectionName).InsertOne(ctx, watcher)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			r.logger.Errorf("failed to insert watcher to db due duplicate error: %s", err)
			return errors.New("watcher already exist")
		}

		r.logger.Errorf("failed to insert watcher to db: %s", err)
		return errors.New("failed to create watcher")
	}

	return nil
}

func (r *repository) UpdateWatcher(ctx context.Context, watcher *Watcher) error {
	_, err := r.db.Database(r.dbName).Collection(r.dbCollectionName).UpdateOne(ctx,
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

	return nil
}

func (r *repository) DeleteWatcher(ctx context.Context, filters bson.M) error {
	_, err := r.db.Database(r.dbName).Collection(r.dbCollectionName).DeleteOne(ctx, filters)
	if err != nil {
		r.logger.Errorf("failed to delete watcher: %s", err)
		return errors.New("failed to delete watcher")
	}

	return nil
}
