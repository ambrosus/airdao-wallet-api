package watcher

import (
	cloudmessaging "airdao-mobile-api/pkg/firebase/cloud-messaging"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

const (
	TOKEN_PRICE_URL = "https://token.ambrosus.io"
)

//go:generate mockgen -source=service.go -destination=mocks/service_mock.go
type Service interface {
	Init(ctx context.Context) error

	TransactionWatch(ctx context.Context, w *Watcher)
	PriceWatch(ctx context.Context, w *Watcher)

	AddAddressWatcher(ctx context.Context, address, pushToken string) error
}

type service struct {
	repository        Repository
	cloudMessagingSvc cloudmessaging.Service
	logger            *zap.SugaredLogger

	cachedPrice float64
}

func NewService(repository Repository, cloudMessagingSvc cloudmessaging.Service, logger *zap.SugaredLogger) (Service, error) {
	if repository == nil {
		return nil, errors.New("[watcher_service] invalid repository")
	}
	if cloudMessagingSvc == nil {
		return nil, errors.New("[watcher_service] cloud messaging service")
	}
	if logger == nil {
		return nil, errors.New("[watcher_service] invalid logger")
	}

	return &service{
		repository:        repository,
		cloudMessagingSvc: cloudMessagingSvc,
		logger:            logger,

		cachedPrice: 0,
	}, nil
}

func (s *service) Init(ctx context.Context) error {
	var priceData *PriceData
	if err := s.doRequest(TOKEN_PRICE_URL, &priceData); err != nil {
		return err
	}

	if priceData != nil {
		s.cachedPrice = priceData.Data.PriceUSD
	}

	page := 1
	for {
		watchers, err := s.repository.GetWatcherList(ctx, bson.M{}, page)
		if err != nil {
			return err
		}

		if watchers == nil {
			break
		}

		for _, w := range watchers {
			go s.TransactionWatch(ctx, w)
			go s.PriceWatch(ctx, w)
		}

		page++
	}

	return nil
}

func (s *service) PriceWatch(ctx context.Context, w *Watcher) {
	for {
		// watch price
		var priceData *PriceData
		if err := s.doRequest(TOKEN_PRICE_URL, &priceData); err != nil {
			s.logger.Errorln(err)
		}

		decodedPushToken, err := base64.StdEncoding.DecodeString(w.PushToken)
		if err != nil {
			s.logger.Errorln(err)
		}

		if priceData != nil {
			increasePercentage := (priceData.Data.PriceUSD - s.cachedPrice) / s.cachedPrice * 100

			if increasePercentage >= 0.01 {
				fmt.Printf("Price increased on %v percent\n", increasePercentage)
				_, err := s.cloudMessagingSvc.SendMessage(ctx, "Price Alert", fmt.Sprintf("Price increased on %v percent\n", increasePercentage), string(decodedPushToken))
				if err != nil {
					s.logger.Errorln(err)
				}
			}

			if increasePercentage <= -0.01 {
				fmt.Printf("Price decrease on %v percent tx\n", increasePercentage)
				response, err := s.cloudMessagingSvc.SendMessage(ctx, "Price Alert", fmt.Sprintf("Price decrease on %v percent tx\n", increasePercentage), string(decodedPushToken))
				if err != nil {
					s.logger.Errorln(err)
				}

				s.logger.Infof("Success sent: %s\n", *response)
			}

			s.cachedPrice = priceData.Data.PriceUSD

		}

		time.Sleep(5 * time.Minute)
	}
}

func (s *service) TransactionWatch(ctx context.Context, w *Watcher) {
	for {
		// watch tx
		var addressData *AddressData
		if err := s.doRequest(fmt.Sprintf("https://explorer-v2-api.ambrosus-test.io/v2/addresses/%s/all", w.Address), &addressData); err != nil {
			s.logger.Errorln(err)
		}

		if addressData != nil && len(addressData.Data) > 0 {

			missedTx := []Tx{}

			for i := range addressData.Data {
				if w.LastTx == nil || (*w.LastTx != addressData.Data[i].Hash && (i+1) < len(addressData.Data) && *w.LastTx == addressData.Data[i+1].Hash) {
					missedTx = append(missedTx, addressData.Data[i])

					w.SetLastTx(missedTx[0].Hash)

					if err := s.repository.UpdateWatcher(ctx, w); err != nil {
						s.logger.Errorln(err)
					}

					fmt.Printf("Your watched address %s have missed %v tx\n", w.Address, len(missedTx))

					decodedPushToken, err := base64.StdEncoding.DecodeString(w.PushToken)
					if err != nil {
						s.logger.Errorln(err)
					}

					response, err := s.cloudMessagingSvc.SendMessage(ctx, "Transaction Alert", fmt.Sprintf("You have new tx: %s", missedTx[0].Hash), string(decodedPushToken))
					if err != nil {
						s.logger.Errorln(err)
					}

					s.logger.Infof("Success sent: %s\n", *response)
					break
				}

				missedTx = append(missedTx, addressData.Data[i])

			}
		}

	}
}

func (s *service) AddAddressWatcher(ctx context.Context, address, pushToken string) error {
	encodePushToken := base64.StdEncoding.EncodeToString([]byte(pushToken))

	dbWatcher, err := s.repository.GetWatcher(ctx, bson.M{"address": address, "push_token": encodePushToken})
	if err != nil {
		return err
	}

	if dbWatcher != nil {
		return errors.New("watcher for this address and token already exist")
	}

	watcher, err := NewWatcher(address, encodePushToken)
	if err != nil {
		return err
	}

	var addressData *AddressData
	if err := s.doRequest(fmt.Sprintf("https://explorer-v2-api.ambrosus-test.io/v2/addresses/%s/all", address), &addressData); err != nil {
		return err
	}

	if addressData != nil && len(addressData.Data) > 0 {
		watcher.SetLastTx(addressData.Data[0].Hash)
	}

	var priceData *PriceData
	if err := s.doRequest(TOKEN_PRICE_URL, &priceData); err != nil {
		return err
	}

	if priceData != nil {
		s.cachedPrice = priceData.Data.PriceUSD
	}

	if err := s.repository.CreateWatcher(ctx, watcher); err != nil {
		return err
	}

	go s.TransactionWatch(ctx, watcher)
	go s.PriceWatch(ctx, watcher)

	return nil
}

func (s *service) doRequest(url string, res interface{}) error {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(respBody, res); err != nil {
		return err
	}

	return nil
}
