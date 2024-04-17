package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	// import watcher repository
	"airdao-mobile-api/services/watcher"
)

// test db name
const dbName = "test_migration"

//type WatcherTestSuite struct {
// 	suite.Suite
// 	pool              *dockertest.Pool
// 	resource          *dockertest.Resource
// 	client            *mongo.Client
// 	watcherRepository Repository
// }
//
// func NewLogger() *zap.SugaredLogger {
// 	logger, err := zap.NewProduction()
// 	if err != nil {
// 		// If error occurs during logger initialization, panic with the error
// 		panic(err)
// 	}
//
// 	// Convert the logger to SugaredLogger for structured, leveled logging
// 	return logger.Sugar()
// }
//
// func (suite *WatcherTestSuite) SetupSuite() {
// 	pool, err := dockertest.NewPool("")
// 	suite.Require().NoError(err)
// 	suite.pool = pool
//
// 	resource, err := pool.Run("mongo", "latest", nil)
// 	suite.Require().NoError(err)
// 	suite.resource = resource
//
// 	var db *mongo.Client
// 	err = pool.Retry(func() error {
// 		uri := fmt.Sprintf("mongodb://localhost:%s", resource.GetPort("27017/tcp"))
// 		client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
// 		if err != nil {
// 			return err
// 		}
// 		db = client
// 		return client.Ping(context.Background(), nil)
// 	})
// 	suite.Require().NoError(err)
//
// 	suite.client = db
//
// 	watcherRepository, err := NewRepository(suite.client, "test", NewLogger())
// 	suite.Require().NoError(err)
// 	suite.watcherRepository = watcherRepository
// }
//
// func (suite *WatcherTestSuite) TearDownSuite() {
// 	suite.Require().NoError(suite.pool.Purge(suite.resource))
// }

// migrtionSuite is a test suite for migration package
type MigrationSuite struct {
	suite.Suite
	pool              *dockertest.Pool
	resource          *dockertest.Resource
	client            *mongo.Client
	watcherRepository watcher.Repository
}

func NewLogger() *zap.SugaredLogger {
	logger, err := zap.NewProduction()
	if err != nil {
		// If error occurs during logger initialization, panic with the error
		panic(err)
	}

	// Convert the logger to SugaredLogger for structured, leveled logging
	return logger.Sugar()
}

// SetupSuite sets up the test suite by initializing the logger
func (suite *MigrationSuite) SetupSuite() {
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
	suite.prepareDB()

	watcherRepository, err := watcher.NewRepository(suite.client, dbName, NewLogger())
	suite.Require().NoError(err)
	suite.watcherRepository = watcherRepository
}

// TearDownSuite tears down the test suite by nullifying the logger

func (suite *MigrationSuite) TearDownSuite() {
	suite.Require().NoError(suite.pool.Purge(suite.resource))
}

// func that prepares db for testing import collection watchers from test_data/watchers.json
func (suite *MigrationSuite) prepareDB() {
	// Open the JSON file
	file, err := os.Open("test_data/watcher.json")
	if err != nil {
		suite.FailNow("Error opening JSON file", err)
	}
	defer file.Close()

	// Read data from the file
	data, err := io.ReadAll(file)
	if err != nil {
		suite.FailNow("Error reading JSON file", err)
	}

	// Define a struct to unmarshal JSON data into
	var watchers []watcher.Watcher // Assuming watcher.Watcher is the type of your data

	//log file str
	//fmt.Println(string(data))

	// Unmarshal JSON data into the struct
	if err := json.Unmarshal(data, &watchers); err != nil {
		suite.FailNow("Error unmarshaling JSON data", err)
	}

	// Get the collection where you want to insert the data
	collection := suite.client.Database(dbName).Collection("watcher")

	// Insert the data into the collection
	for _, w := range watchers {
		//skip in history is more than 1000
		if w.HistoricalNotifications != nil && len(*w.HistoricalNotifications) > 1000 {
			continue
		}
		if _, err := collection.InsertOne(context.Background(), w); err != nil {
			suite.FailNow("Error inserting data into collection", err)
		}
	}
}

//
//// func that prepares db for testing import collection watchers from test_data/watcher.json
//func (suite *MigrationSuite) prepareDB() {
//	// Open the BSON file
//	file, err := os.Open("test_data/watcher.bson")
//	if err != nil {
//		suite.FailNow("Error opening BSON file", err)
//	}
//	defer file.Close()
//
//	// Read data from the file
//	// Note: BSON data doesn't need to be decoded like JSON
//	// Instead, we directly read it into a byte slice
//	data, err := io.ReadAll(file)
//	if err != nil {
//		suite.FailNow("Error reading BSON file", err)
//	}
//
//	fmt.Println("Data read successfully")
//	// Convert BSON data to BSON documents
//	var documents []bson.M
//	err = bson.Unmarshal(data, &documents)
//	if err != nil {
//		suite.FailNow("Error unmarshaling BSON data", err)
//	}
//
//	// Get the collection where you want to insert the data
//	collection := suite.client.Database(dbName).Collection("watcher")
//
//	// Insert the data into the collection
//	for _, doc := range documents {
//		if _, err := collection.InsertOne(context.Background(), doc); err != nil {
//			suite.FailNow("Error inserting data into collection", err)
//		}
//	}
//	fmt.Println("Data inserted successfully")
//}

func (suite *MigrationSuite) TestMigration() {
	err := RunMigrations(suite.client, dbName, NewLogger())
	suite.Require().NoError(err)
	err = suite.watcherRepository.DeleteWatchersWithStaleData(context.Background())
	suite.Require().NoError(err)
	// test all the functions from watcher repository and watcher for example SetLastTx

}

func TestMigrationSuite(t *testing.T) {
	suite.Run(t, new(MigrationSuite))
}
