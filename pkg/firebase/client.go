package firebase

import (
	"context"
	"errors"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

//go:generate mockgen -source=client.go -destination=mocks/client_mock.go
type Client interface {
	CreateCloudMessagingClient() (*messaging.Client, error)
}

type client struct {
	credentialsPath string
}

func NewClient(credentialsPath string) (Client, error) {
	if credentialsPath == "" {
		return nil, errors.New("invalid firebase credentials path")
	}

	return &client{credentialsPath: credentialsPath}, nil
}

func (c *client) CreateCloudMessagingClient() (*messaging.Client, error) {
	opts := []option.ClientOption{option.WithCredentialsFile(c.credentialsPath)}

	app, err := firebase.NewApp(context.Background(), nil, opts...)
	if err != nil {
		return nil, err
	}

	fcmClient, err := app.Messaging(context.Background())
	if err != nil {
		return nil, err
	}

	return fcmClient, nil
}
