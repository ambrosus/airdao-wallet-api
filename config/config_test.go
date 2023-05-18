package config_test

import (
	"os"
	"reflect"
	"testing"

	"airdao-mobile-api/config"
)

func TestInit(t *testing.T) {
	type env struct {
		environment      string
		port             string
		mongoDbName      string
		mongoDbUrl       string
		firebaseCredPath string
		androidChannel   string
	}

	type args struct {
		env env
	}

	setEnv := func(env env) {
		os.Setenv("APP_ENV", env.environment)
		os.Setenv("PORT", env.port)
		os.Setenv("MONGO_DB_NAME", env.mongoDbName)
		os.Setenv("MONGO_DB_URL", env.mongoDbUrl)
		os.Setenv("FIREBASE_CRED_PATH", env.firebaseCredPath)
		os.Setenv("ANDROID_CHANNEL_NAME", env.androidChannel)
	}

	tests := []struct {
		name      string
		args      args
		want      *config.Config
		wantError bool
	}{
		{
			name: "Test config file!",
			args: args{
				env: env{
					environment:      "development",
					port:             "5000",
					mongoDbName:      "example",
					mongoDbUrl:       "http://127.0.0.1",
					firebaseCredPath: "./example.json",
					androidChannel:   "example",
				},
			},
			want: &config.Config{
				Environment: "development",
				Port:        "5000",
				MongoDb: config.MongoDb{
					MongoDbName: "example",
					MongoDbUrl:  "http://127.0.0.1",
				},
				Firebase: config.Firebase{
					CredPath:           "./example.json",
					AndroidChannelName: "example",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setEnv(test.args.env)

			got, err := config.GetConfig()
			if (err != nil) != test.wantError {
				t.Errorf("Init() error = %s, wantErr %v", err, test.wantError)

				return
			}

			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Init() got = %v, want %v", got, test.want)
			}
		})
	}
}
