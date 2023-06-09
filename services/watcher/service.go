package watcher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	cloudmessaging "airdao-mobile-api/pkg/firebase/cloud-messaging"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

const (
	TOKEN_PRICE_URL = "https://token.ambrosus.io"
)

//go:generate mockgen -source=service.go -destination=mocks/service_mock.go
type Service interface {
	Init(ctx context.Context) error

	TransactionWatch(ctx context.Context, w *Watcher, stopChan chan struct{})
	ApiPriceWatch(ctx context.Context)
	PriceWatch(ctx context.Context, w *Watcher, stopChan chan struct{})

	CreateWatcher(ctx context.Context, address, pushToken string, threshold int) error
	UpdateWatcher(ctx context.Context, address, pushToken string, threshold int) error
	DeleteWatcher(ctx context.Context, address, pushToken string) error
}

type service struct {
	repository        Repository
	cloudMessagingSvc cloudmessaging.Service
	logger            *zap.SugaredLogger

	mx               sync.RWMutex
	cachedChan       map[string]chan struct{}
	cachedPrice      float64
	cachedPercentage float64
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

		cachedChan:       make(map[string]chan struct{}),
		cachedPrice:      0,
		cachedPercentage: 0,
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
			s.sentUpStopChanAndStartWatchers(ctx, w)
		}

		page++
	}

	go s.ApiPriceWatch(ctx)

	return nil
}

func (s *service) ApiPriceWatch(ctx context.Context) {
	for {
		var priceData *PriceData
		if err := s.doRequest(TOKEN_PRICE_URL, &priceData); err != nil {
			s.logger.Errorln(err)
		}

		if priceData != nil {
			percentage := (priceData.Data.PriceUSD - s.cachedPrice) / s.cachedPrice * 100

			s.cachedPrice = priceData.Data.PriceUSD
			s.cachedPercentage = percentage
		}

		// time.Sleep(1 * time.Minute)
		time.Sleep(5 * time.Minute)
	}
}

func (s *service) PriceWatch(ctx context.Context, w *Watcher, stopChan chan struct{}) {
	for {
		select {
		case <-stopChan:
			// fmt.Println("Stopping price goroutine...")
			return
		default:
			decodedPushToken, err := base64.StdEncoding.DecodeString(w.PushToken)
			if err != nil {
				s.logger.Errorln(err)
			}

			if s.cachedPercentage >= float64(*w.Threshold) {
				data := map[string]interface{}{"type": "price-alert", "percentage": s.cachedPercentage}

				response, err := s.cloudMessagingSvc.SendMessage(ctx, "Price Alert", fmt.Sprintf("Price increased on %v percent\n", s.cachedPercentage), string(decodedPushToken), data)
				if err != nil {
					s.logger.Errorln(err)
				}

				if response != nil {
					s.logger.Infof("Price notification successfully sent")
				}
			}

			if s.cachedPercentage <= -float64(*w.Threshold) {
				data := map[string]interface{}{"type": "price-alert", "percentage": s.cachedPercentage}

				response, err := s.cloudMessagingSvc.SendMessage(ctx, "Price Alert", fmt.Sprintf("Price decrease on %v percent tx\n", s.cachedPercentage), string(decodedPushToken), data)
				if err != nil {
					s.logger.Errorln(err)
				}

				if response != nil {
					s.logger.Infof("Price notification successfully sent")
				}
			}

			time.Sleep(330 * time.Second)
			// time.Sleep(90 * time.Second)
		}
	}
}

func (s *service) TransactionWatch(ctx context.Context, w *Watcher, stopChan chan struct{}) {
	for {
		select {
		case <-stopChan:
			// fmt.Println("Stopping transaction goroutine...")
			return
		default:
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

						data := map[string]interface{}{"type": "transaction-alert"}

						decodedPushToken, err := base64.StdEncoding.DecodeString(w.PushToken)
						if err != nil {
							s.logger.Errorln(err)
						}

						response, err := s.cloudMessagingSvc.SendMessage(ctx, "Transaction Alert", fmt.Sprintf("You have new tx: %s", missedTx[0].Hash), string(decodedPushToken), data)
						if err != nil {
							s.logger.Errorln(err)
						}

						if response != nil {
							s.logger.Infof("Transaction notification successfully sent")
						}

						break
					}

					missedTx = append(missedTx, addressData.Data[i])

				}
			}

			time.Sleep(10 * time.Second)
		}
	}
}

func (s *service) CreateWatcher(ctx context.Context, address, pushToken string, threshold int) error {
	encodePushToken := base64.StdEncoding.EncodeToString([]byte(pushToken))

	dbWatcher, err := s.repository.GetWatcher(ctx, bson.M{"address": address, "push_token": encodePushToken})
	if err != nil {
		return err
	}

	if dbWatcher != nil {
		return errors.New("watcher for this address and token already exist")
	}

	watcher, err := NewWatcher(address, encodePushToken, &threshold)
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

	s.sentUpStopChanAndStartWatchers(ctx, watcher)

	return nil
}

func (s *service) UpdateWatcher(ctx context.Context, address, pushToken string, threshold int) error {
	encodePushToken := base64.StdEncoding.EncodeToString([]byte(pushToken))

	watcher, err := s.repository.GetWatcher(ctx, bson.M{"address": address, "push_token": encodePushToken})
	if err != nil {
		return err
	}

	if threshold != *watcher.Threshold {
		watcherStopChan := s.cachedChan[watcher.ID.Hex()]
		close(watcherStopChan)
		delete(s.cachedChan, watcher.ID.Hex())

		watcher.SetThreshold(threshold)

		if err := s.repository.UpdateWatcher(ctx, watcher); err != nil {
			return err
		}

		s.sentUpStopChanAndStartWatchers(ctx, watcher)
	}

	return nil
}

func (s *service) DeleteWatcher(ctx context.Context, address, pushToken string) error {
	encodePushToken := base64.StdEncoding.EncodeToString([]byte(pushToken))

	filters := bson.M{"address": address, "push_token": encodePushToken}

	dbWatcher, err := s.repository.GetWatcher(ctx, filters)
	if err != nil {
		return err
	}

	if dbWatcher == nil {
		return errors.New("watcher not found")
	}

	if err := s.repository.DeleteWatcher(ctx, filters); err != nil {
		return err
	}

	s.mx.RLock()
	defer s.mx.RUnlock()

	watcherStopChan := s.cachedChan[dbWatcher.ID.Hex()]
	close(watcherStopChan)
	delete(s.cachedChan, dbWatcher.ID.Hex())

	return nil
}

func (s *service) sentUpStopChanAndStartWatchers(ctx context.Context, w *Watcher) {
	s.mx.Lock()
	defer s.mx.Unlock()

	stopChan := make(chan struct{})
	s.cachedChan[w.ID.Hex()] = stopChan

	go s.TransactionWatch(ctx, w, stopChan)
	go s.PriceWatch(ctx, w, stopChan)
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
