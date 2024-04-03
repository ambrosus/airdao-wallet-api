package watcher

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type WatcherTestSuite struct {
	suite.Suite
	pool              *dockertest.Pool
	resource          *dockertest.Resource
	client            *mongo.Client
	watcherRepository Repository
}

func (suite *WatcherTestSuite) SetupSuite() {
	pool, err := dockertest.NewPool("")
	suite.Require().NoError(err)
	suite.pool = pool

	resource, err := pool.Run("mongo", "latest", nil)
	suite.Require().NoError(err)
	suite.resource = resource

	var db *mongo.Client
	err = pool.Retry(func() error {
		uri := fmt.Sprintf("mongodb://localhost:%s", resource.GetPort("27017/tcp"))
		client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
		if err != nil {
			return err
		}
		db = client
		return client.Ping(context.Background(), nil)
	})
	suite.Require().NoError(err)

	suite.client = db

	logger := zap.Logger{}
	sugaredLog := logger.Sugar()
	watcherRepository, err := NewRepository(suite.client, "test", sugaredLog)
	suite.Require().NoError(err)
	suite.watcherRepository = watcherRepository
}

func (suite *WatcherTestSuite) TearDownSuite() {
	suite.Require().NoError(suite.pool.Purge(suite.resource))
}

func (suite *WatcherTestSuite) TestCreateWatcher() {
	// Test create watcher and verify it's saved in the repository
	pushToken := "testPushToken"
	watcher, err := NewWatcher(pushToken)
	suite.Require().NoError(err)

	ctx := context.Background()
	err = suite.watcherRepository.CreateWatcher(ctx, watcher)
	suite.Require().NoError(err)
}

func (suite *WatcherTestSuite) TestReadWatcher() {
	// Test read watcher by ID and verify its properties

	// Create a new watcher
	pushToken := "testPushToken"
	watcher, err := NewWatcher(pushToken)
	suite.Require().NoError(err)

	//add historical notifications
	timestampNowUtc := time.Now().UTC()
	watcher.AddNotification("title", "body", true, timestampNowUtc)
	watcher.AddNotification("title2", "body2", true, timestampNowUtc)

	// Create the watcher in the repository
	ctx := context.Background()
	err = suite.watcherRepository.CreateWatcher(ctx, watcher)
	suite.Require().NoError(err)

	// Read the watcher by ID
	filter := bson.M{"_id": watcher.ID}
	resultWatcher, err := suite.watcherRepository.GetWatcher(ctx, filter)
	suite.Require().NoError(err)
	suite.Require().NotNil(resultWatcher)
	suite.Require().Equal(pushToken, resultWatcher.PushToken)
	// assert bson equality
	bsonResultWatcher, err := bson.Marshal(resultWatcher)
	suite.Require().NoError(err)
	bsonResultWatcherString := string(bsonResultWatcher)
	bsonWatcher, err := bson.Marshal(watcher)
	suite.Require().NoError(err)
	bsonWatcherString := string(bsonWatcher)
	suite.Require().Equal(bsonWatcherString, bsonResultWatcherString)
}

func (suite *WatcherTestSuite) TestUpdateWatcher() {
	// Test update watcher and verify the changes are saved in the repository

	// Create a new watcher
	pushToken := "testPushToken"
	watcher, err := NewWatcher(pushToken)
	suite.Require().NoError(err)

	//CreateWatcher

	ctx := context.Background()
	err = suite.watcherRepository.CreateWatcher(ctx, watcher)
	suite.Require().NoError(err)

	// Update the watcher
	newPushToken := "newTestPushToken"
	watcher.PushToken = newPushToken
	timestampNowUtc := time.Now().UTC()
	watcher.AddNotification("title", "body", true, timestampNowUtc)
	watcher.AddNotification("title2", "body2", true, timestampNowUtc)
	err = suite.watcherRepository.UpdateWatcher(ctx, watcher)
	suite.Require().NoError(err)

	// Read the updated watcher
	filter := bson.M{"_id": watcher.ID}
	resultWatcher, err := suite.watcherRepository.GetWatcher(ctx, filter)
	suite.Require().NoError(err)
	suite.Require().NotNil(resultWatcher)
	suite.Require().Equal(newPushToken, resultWatcher.PushToken)
	// assert bson equality
	bsonResultWatcher, err := bson.Marshal(resultWatcher)
	suite.Require().NoError(err)
	bsonResultWatcherString := string(bsonResultWatcher)
	bsonWatcher, err := bson.Marshal(watcher)
	suite.Require().NoError(err)
	bsonWatcherString := string(bsonWatcher)
	suite.Require().Equal(bsonWatcherString, bsonResultWatcherString)
}

func (suite *WatcherTestSuite) TestDeleteWatcher() {
	// Test delete watcher and verify it's removed from the repository

	// Create a new watcher
	pushToken := "deleteTestPushToken"
	watcher, err := NewWatcher(pushToken)
	suite.Require().NoError(err)

	// Create the watcher in the repository
	ctx := context.Background()
	err = suite.watcherRepository.CreateWatcher(ctx, watcher)
	suite.Require().NoError(err)

	// Delete the watcher
	filter := bson.M{"_id": watcher.ID}
	err = suite.watcherRepository.DeleteWatcher(ctx, filter)
	suite.Require().NoError(err)

	// Read all watchers
	watchers, err := suite.watcherRepository.GetAllWatchers(ctx)
	suite.Require().NoError(err)
	suite.Require().NotNil(watchers)
	// Verify that the watcher has been deleted find the watcher by ID
	found := false
	for _, w := range watchers {
		if w.ID == watcher.ID {
			found = true
			break
		}
	}
	suite.Require().False(found)
}

func (suite *WatcherTestSuite) TestWatcherWithHistoricalNotifications() {
	// Test watcher with historical notifications and verify they are saved and retrieved correctly

	// Create a new watcher
	pushToken := "testPushToken"
	watcher, err := NewWatcher(pushToken)
	suite.Require().NoError(err)

	// Add historical notifications
	timestampNowUtc := time.Now().UTC()
	watcher.AddNotification("title", "body", true, timestampNowUtc)
	watcher.AddNotification("title2", "body2", true, timestampNowUtc)

	// Create the watcher in the repository
	ctx := context.Background()
	err = suite.watcherRepository.CreateWatcher(ctx, watcher)
	suite.Require().NoError(err)

	// Read the watcher by ID
	filter := bson.M{"_id": watcher.ID}
	resultWatcher, err := suite.watcherRepository.GetWatcher(ctx, filter)
	suite.Require().NoError(err)
	suite.Require().NotNil(resultWatcher)

	// Verify the historical notifications
	suite.Require().Equal(2, len(*resultWatcher.HistoricalNotifications))

	// assert bson equality
	bsonResultWatcher, err := bson.Marshal(resultWatcher)
	suite.Require().NoError(err)
	bsonResultWatcherString := string(bsonResultWatcher)
	bsonWatcher, err := bson.Marshal(watcher)
	suite.Require().NoError(err)
	bsonWatcherString := string(bsonWatcher)
	suite.Require().Equal(bsonWatcherString, bsonResultWatcherString)
}

func (suite *WatcherTestSuite) TestWatcherWithAddressAndNotifications() {
	// Test watcher with addresses and notifications and verify they are saved and retrieved correctly

	// Create a new watcher
	pushToken := "testPushToken"
	watcher, err := NewWatcher(pushToken)
	suite.Require().NoError(err)
	// Create the watcher in the repository
	ctx := context.Background()
	err = suite.watcherRepository.CreateWatcher(ctx, watcher)
	suite.Require().NoError(err)

	// Add addresses
	watcher.AddAddress("0x1234567890")
	watcher.AddAddress("0x0987654321")

	// Add notifications
	timestampNowUtc := time.Now().UTC()
	watcher.AddNotification("title", "body", true, timestampNowUtc)
	watcher.AddNotification("title2", "body2", true, timestampNowUtc)

	// Update the watcher
	threshold := 10.0
	txNotification := "on"
	priceNotification := "on"
	watcher.SetThreshold(threshold)
	watcher.SetTxNotification(txNotification)
	watcher.SetPriceNotification(priceNotification)
	watcher.SetLastSuccessDate(time.Now())
	watcher.SetLastFailDate(time.Now())
	watcher.SetTokenPrice(100.0)
	watcher.SetLastTx("0x1234567890", "0x0987654321")
	err = suite.watcherRepository.UpdateWatcher(ctx, watcher)
	suite.Require().NoError(err)

	// Read the watcher by ID
	filter := bson.M{"_id": watcher.ID}
	resultWatcher, err := suite.watcherRepository.GetWatcher(ctx, filter)
	suite.Require().NoError(err)
	suite.Require().NotNil(resultWatcher)

	// Verify the addresses
	suite.Require().Equal(2, len(*resultWatcher.Addresses))

	// Verify the notifications
	suite.Require().Equal(2, len(*resultWatcher.HistoricalNotifications))

	// assert bson equality
	bsonResultWatcher, err := bson.Marshal(resultWatcher)
	suite.Require().NoError(err)
	bsonResultWatcherString := string(bsonResultWatcher)
	bsonWatcher, err := bson.Marshal(watcher)
	suite.Require().NoError(err)
	bsonWatcherString := string(bsonWatcher)
	suite.Require().Equal(bsonWatcherString, bsonResultWatcherString)

}

func TestWatcherSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}
