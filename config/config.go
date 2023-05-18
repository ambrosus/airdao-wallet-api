package config

import (
	"sync"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Environment string `required:"true" envconfig:"APP_ENV"`
	Port        string `required:"true" envconfig:"PORT"`

	MongoDb
	Firebase
}

type MongoDb struct {
	MongoDbName string `required:"true" envconfig:"MONGO_DB_NAME"`
	MongoDbUrl  string `required:"true" envconfig:"MONGO_DB_URL"`
}

type Firebase struct {
	CredPath           string `required:"true" envconfig:"FIREBASE_CRED_PATH"`
	AndroidChannelName string `required:"true" envconfig:"ANDROID_CHANNEL_NAME"`
}

var (
	once   sync.Once
	config *Config
)

func GetConfig() (*Config, error) {
	var err error
	once.Do(func() {
		var cfg Config
		_ = godotenv.Load(".env")

		if err = envconfig.Process("", &cfg); err != nil {
			return
		}

		config = &cfg
	})

	return config, err
}
