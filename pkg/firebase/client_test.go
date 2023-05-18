package firebase_test

import (
	"airdao-mobile-api/pkg/firebase"
	"encoding/json"

	"os"
	"path/filepath"
	"testing"

	"firebase.google.com/go/messaging"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {

	path := "./path.json"

	type args struct {
		credentialsPath string
	}
	tests := []struct {
		name   string
		args   args
		expect func(t *testing.T, c firebase.Client, err error)
	}{
		{
			name: "should return client",
			args: args{
				credentialsPath: path,
			},
			expect: func(t *testing.T, c firebase.Client, err error) {
				assert.NotNil(t, c)
				assert.Nil(t, err)
			},
		},
		{
			name: "should return invalid firebase credentials path",
			args: args{
				credentialsPath: "",
			},
			expect: func(t *testing.T, c firebase.Client, err error) {
				assert.Nil(t, c)
				assert.NotNil(t, err)
				assert.EqualError(t, err, "invalid firebase credentials path")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := firebase.NewClient(tc.args.credentialsPath)
			tc.expect(t, got, err)
		})
	}
}

func TestCreateCloudMessagingClient(t *testing.T) {

	testCred := struct {
		Type                    string `json:"type"`
		Projectid               string `json:"project_id"`
		PrivateKeyId            string `json:"private_key_id"`
		PrivateKey              string `json:"private_key"`
		ClientEmail             string `json:"client_email"`
		ClientId                string `json:"client_id"`
		AuthUri                 string `json:"auth_uri"`
		TokenUri                string `json:"token_uri"`
		AuthProviderX509CertUrl string `json:"auth_provider_x509_cert_url"`
		ClientX509CertUrl       string `json:"client_x509_cert_url"`
	}{
		Type:                    "service_account",
		Projectid:               "example",
		PrivateKeyId:            "example",
		PrivateKey:              "example",
		ClientEmail:             "example",
		ClientId:                "example",
		AuthUri:                 "example",
		TokenUri:                "example",
		AuthProviderX509CertUrl: "example",
		ClientX509CertUrl:       "example",
	}

	d, _ := json.Marshal(testCred)

	f, _ := os.Create("cred.json")
	f.Write(d)

	path, _ := filepath.Abs("cred.json")

	type fields struct {
		credentialsPath string
	}
	tests := []struct {
		name   string
		fields fields
		expect func(t *testing.T, mc *messaging.Client, err error)
	}{
		{
			name: "should return cloud messaging client",
			fields: fields{
				credentialsPath: path,
			},
			expect: func(t *testing.T, mc *messaging.Client, err error) {
				assert.NotNil(t, mc)
				assert.Nil(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := firebase.NewClient(tc.fields.credentialsPath)
			got, err := c.CreateCloudMessagingClient()
			tc.expect(t, got, err)
		})

		_ = os.Remove(path)
	}
}
