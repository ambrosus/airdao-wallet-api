package cloudmessaging

import (
	"context"
	"errors"

	"firebase.google.com/go/messaging"
)

type Service interface {
	SendMessage(ctx context.Context, title, body, pushToken string) (*string, error)
}

type service struct {
	fcmClient      *messaging.Client
	androidChannel string
}

func NewCloudMessagingService(fcmClient *messaging.Client, androidChannel string) (Service, error) {
	if fcmClient == nil {
		return nil, errors.New("invalid firebase messaging client")
	}
	if androidChannel == "" {
		return nil, errors.New("invalid firebase android channel")
	}

	return &service{fcmClient: fcmClient, androidChannel: androidChannel}, nil
}

func (s *service) SendMessage(ctx context.Context, title, body, pushToken string) (*string, error) {
	response, err := s.fcmClient.Send(ctx, &messaging.Message{

		// iOS
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: title,
						Body:  body,
					},
					Sound: "default",
				},
			},
		},

		// Android
		Android: &messaging.AndroidConfig{
			Notification: &messaging.AndroidNotification{
				Title:     title,
				Body:      body,
				ChannelID: s.androidChannel,
				Sound:     "default",
			},
		},
		Token: pushToken,
	})

	if err != nil {
		return nil, err
	}

	return &response, err
}
