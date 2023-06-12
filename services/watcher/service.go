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

	TransactionWatch(ctx context.Context, watcherId string, stopChan chan struct{})
	ApiPriceWatch(ctx context.Context)
	PriceWatch(ctx context.Context, watcherId string, stopChan chan struct{})

	CreateWatcher(ctx context.Context, pushToken string) error
	UpdateWatcher(ctx context.Context, pushToken string, address *[]string, threshold *int) error
	DeleteWatcher(ctx context.Context, pushToken string) error
}

type service struct {
	repository        Repository
	cloudMessagingSvc cloudmessaging.Service
	logger            *zap.SugaredLogger

	mx               sync.RWMutex
	cachedWatcher    map[string]*Watcher
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
		cachedWatcher:    make(map[string]*Watcher),
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

		for _, watcher := range watchers {
			s.cachedWatcher[watcher.ID.Hex()] = watcher

			s.sentUpStopChanAndStartWatchers(ctx, watcher)
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

func (s *service) PriceWatch(ctx context.Context, watcherId string, stopChan chan struct{}) {
	for {
		select {
		case <-stopChan:
			// fmt.Println("Stopping price goroutine...")
			return
		default:
			watcher, ok := s.cachedWatcher[watcherId]
			if !ok {
				continue
			}

			if watcher != nil && watcher.Threshold != nil {
				decodedPushToken, err := base64.StdEncoding.DecodeString(watcher.PushToken)
				if err != nil {
					s.logger.Errorln(err)
				}

				if s.cachedPercentage >= float64(*watcher.Threshold) {
					data := map[string]interface{}{"type": "price-alert", "percentage": s.cachedPercentage}

					response, err := s.cloudMessagingSvc.SendMessage(ctx, "Price Alert", fmt.Sprintf("Price increased on %v percent\n", s.cachedPercentage), string(decodedPushToken), data)
					if err != nil {
						s.logger.Errorln(err)
					}

					if response != nil {
						s.logger.Infof("Price notification successfully sent")
					}
				}

				if s.cachedPercentage <= -float64(*watcher.Threshold) {
					data := map[string]interface{}{"type": "price-alert", "percentage": s.cachedPercentage}

					response, err := s.cloudMessagingSvc.SendMessage(ctx, "Price Alert", fmt.Sprintf("Price decrease on %v percent tx\n", s.cachedPercentage), string(decodedPushToken), data)
					if err != nil {
						s.logger.Errorln(err)
					}

					if response != nil {
						s.logger.Infof("Price notification successfully sent")
					}
				}
			}

			time.Sleep(330 * time.Second)
			// time.Sleep(90 * time.Second)
		}
	}
}

func (s *service) TransactionWatch(ctx context.Context, watcherId string, stopChan chan struct{}) {
	for {
		select {
		case <-stopChan:
			// fmt.Println("Stopping transaction goroutine...")
			return
		default:
			s.mx.RLock()
			watcher, ok := s.cachedWatcher[watcherId]
			if !ok {
				continue
			}
			s.mx.RUnlock()

			if watcher != nil && (watcher.Addresses != nil && len(*watcher.Addresses) > 0) {
				for _, address := range *watcher.Addresses {
					var apiAddressData *ApiAddressData
					if err := s.doRequest(fmt.Sprintf("https://explorer-v2-api.ambrosus-test.io/v2/addresses/%s/all", address.Address), &apiAddressData); err != nil {
						s.logger.Errorln(err)
					}

					if apiAddressData != nil && len(apiAddressData.Data) > 0 {

						missedTx := []Tx{}

						for i := range apiAddressData.Data {
							if address.LastTx == nil || (*address.LastTx != apiAddressData.Data[i].Hash && (i+1) < len(apiAddressData.Data) && *address.LastTx == apiAddressData.Data[i+1].Hash) {
								missedTx = append(missedTx, apiAddressData.Data[i])

								watcher.SetLastTx(address.Address, missedTx[0].Hash)

								if err := s.repository.UpdateWatcher(ctx, watcher); err != nil {
									s.logger.Errorln(err)
								}

								s.mx.Lock()
								s.cachedWatcher[watcher.ID.Hex()] = watcher
								s.mx.Unlock()

								data := map[string]interface{}{"type": "transaction-alert"}

								decodedPushToken, err := base64.StdEncoding.DecodeString(watcher.PushToken)
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

							missedTx = append(missedTx, apiAddressData.Data[i])

						}
					}
				}
			}

			time.Sleep(10 * time.Second)
		}
	}
}

func (s *service) CreateWatcher(ctx context.Context, pushToken string) error {
	encodePushToken := base64.StdEncoding.EncodeToString([]byte(pushToken))
	dbWatcher, err := s.repository.GetWatcher(ctx, bson.M{"push_token": encodePushToken})
	if err != nil {
		return err
	}

	if dbWatcher != nil {
		return errors.New("watcher for this address and token already exist")
	}

	watcher, err := NewWatcher(encodePushToken)
	if err != nil {
		return err
	}

	if err := s.repository.CreateWatcher(ctx, watcher); err != nil {
		return err
	}

	s.cachedWatcher[watcher.ID.Hex()] = watcher

	s.sentUpStopChanAndStartWatchers(ctx, watcher)

	return nil
}

func (s *service) UpdateWatcher(ctx context.Context, pushToken string, addresses *[]string, threshold *int) error {
	encodePushToken := base64.StdEncoding.EncodeToString([]byte(pushToken))

	watcher, err := s.repository.GetWatcher(ctx, bson.M{"push_token": encodePushToken})
	if err != nil {
		return err
	}

	if watcher == nil {
		return errors.New("watcher not found")
	}

	if addresses != nil {
		for _, address := range *addresses {

			if watcher.Addresses != nil {
				for _, v := range *watcher.Addresses {
					if address == v.Address {
						return errors.New("address already is watched")
					}
				}
			}

			watcher.AddAddress(address)

			var apiAddressData *ApiAddressData
			if err := s.doRequest(fmt.Sprintf("https://explorer-v2-api.ambrosus-test.io/v2/addresses/%s/all", address), &apiAddressData); err != nil {
				s.logger.Errorln(err)
			}

			if apiAddressData != nil && len(apiAddressData.Data) > 0 {
				watcher.SetLastTx(address, apiAddressData.Data[0].Hash)
			}
		}
	}

	if threshold != nil && (watcher.Threshold == nil || *threshold != *watcher.Threshold) {
		watcher.SetThreshold(*threshold)
	}

	if err := s.repository.UpdateWatcher(ctx, watcher); err != nil {
		return err
	}

	s.mx.Lock()
	s.cachedWatcher[watcher.ID.Hex()] = watcher
	s.mx.Unlock()

	return nil
}

func (s *service) DeleteWatcher(ctx context.Context, pushToken string) error {
	encodePushToken := base64.StdEncoding.EncodeToString([]byte(pushToken))

	filters := bson.M{"push_token": encodePushToken}

	watcher, err := s.repository.GetWatcher(ctx, filters)
	if err != nil {
		return err
	}

	if watcher == nil {
		return errors.New("watcher not found")
	}

	if err := s.repository.DeleteWatcher(ctx, filters); err != nil {
		return err
	}

	s.mx.RLock()
	watcherStopChan := s.cachedChan[watcher.ID.Hex()]
	s.mx.RUnlock()

	close(watcherStopChan)
	delete(s.cachedWatcher, watcher.ID.Hex())
	delete(s.cachedChan, watcher.ID.Hex())

	return nil
}

func (s *service) sentUpStopChanAndStartWatchers(ctx context.Context, watcher *Watcher) {
	s.mx.Lock()
	defer s.mx.Unlock()

	stopChan := make(chan struct{})
	s.cachedChan[watcher.ID.Hex()] = stopChan

	go s.TransactionWatch(ctx, watcher.ID.Hex(), stopChan)
	go s.PriceWatch(ctx, watcher.ID.Hex(), stopChan)
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
