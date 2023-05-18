package cloudmessaging_test

import (
	cloudmessaging "airdao-mobile-api/pkg/firebase/cloud-messaging"
	"testing"

	"firebase.google.com/go/messaging"
	"github.com/stretchr/testify/assert"
)

func TestNewCloudMessagingService(t *testing.T) {

	androidChannelName := "channel"

	type args struct {
		fcmClient          *messaging.Client
		androidChannelName string
	}
	tests := []struct {
		name   string
		args   args
		expect func(t *testing.T, s cloudmessaging.Service, err error)
	}{
		{
			name: "should return service",
			args: args{
				fcmClient:          &messaging.Client{},
				androidChannelName: androidChannelName,
			},
			expect: func(t *testing.T, s cloudmessaging.Service, err error) {
				assert.NotNil(t, s)
				assert.Nil(t, err)
			},
		},
	}
	for _, tс := range tests {
		t.Run(tс.name, func(t *testing.T) {
			got, err := cloudmessaging.NewCloudMessagingService(tс.args.fcmClient, tс.args.androidChannelName)
			tс.expect(t, got, err)
		})
	}
}
