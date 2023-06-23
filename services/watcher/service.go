package watcher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	cloudmessaging "airdao-mobile-api/pkg/firebase/cloud-messaging"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

const (
	ON  = "on"
	OFF = "off"
)

//go:generate mockgen -source=service.go -destination=mocks/service_mock.go
type Service interface {
	Init(ctx context.Context) error

	TransactionWatch(ctx context.Context, watcherId string, stopChan chan struct{})
	ApiPriceWatch(ctx context.Context)
	PriceWatch(ctx context.Context, watcherId string, stopChan chan struct{})
	CGWatch(ctx context.Context)

	GetWatcher(ctx context.Context, pushToken string) (*Watcher, error)
	GetWatcherHistoryPrices(ctx context.Context) *CGData
	CreateWatcher(ctx context.Context, pushToken string) error
	UpdateWatcher(ctx context.Context, pushToken string, addresses *[]string, threshold *float64, txNotification, priceNotification *string) error
	DeleteWatcher(ctx context.Context, pushToken string) error
	DeleteWatcherAddresses(ctx context.Context, pushToken string, addresses []string) error
}

type service struct {
	repository        Repository
	cloudMessagingSvc cloudmessaging.Service
	logger            *zap.SugaredLogger

	explorerUrl   string
	tokenPriceUrl string

	mx            sync.RWMutex
	cachedWatcher map[string]*Watcher
	cachedChan    map[string]chan struct{}
	cachedPrice   float64
	cachedCgPrice [][]float64
}

func NewService(
	repository Repository,
	cloudMessagingSvc cloudmessaging.Service,
	logger *zap.SugaredLogger,
	explorerUrl string,
	tokenPriceUrl string) (Service, error) {
	if repository == nil {
		return nil, errors.New("[watcher_service] invalid repository")
	}
	if cloudMessagingSvc == nil {
		return nil, errors.New("[watcher_service] cloud messaging service")
	}
	if logger == nil {
		return nil, errors.New("[watcher_service] invalid logger")
	}
	if explorerUrl == "" {
		return nil, errors.New("[watcher_service] invalid explorer url")
	}
	if tokenPriceUrl == "" {
		return nil, errors.New("[watcher_service] invalid token price url")
	}

	return &service{
		repository:        repository,
		cloudMessagingSvc: cloudMessagingSvc,
		logger:            logger,

		explorerUrl:   explorerUrl,
		tokenPriceUrl: tokenPriceUrl,

		cachedChan:    make(map[string]chan struct{}),
		cachedWatcher: make(map[string]*Watcher),
		cachedPrice:   0,
		cachedCgPrice: [][]float64{},
	}, nil
}

func (s *service) Init(ctx context.Context) error {
	var priceData *PriceData
	if err := s.doRequest(s.tokenPriceUrl, &priceData); err != nil {
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
			s.mx.Lock()
			s.cachedWatcher[watcher.ID.Hex()] = watcher
			s.mx.Unlock()

			s.sentUpStopChanAndStartWatchers(ctx, watcher)
		}

		page++
	}

	go s.CGWatch(ctx)
	go s.ApiPriceWatch(ctx)

	return nil
}

func (s *service) CGWatch(ctx context.Context) {
	for {
		var cgData *CGData
		if err := s.doRequest("https://api.coingecko.com/api/v3/coins/amber/market_chart?vs_currency=usd&days=30", &cgData); err != nil {
			s.logger.Errorln(err)
		}

		s.cachedCgPrice = cgData.Prices

		time.Sleep(12 * time.Hour)
	}
}

func (s *service) ApiPriceWatch(ctx context.Context) {
	for {
		var priceData *PriceData
		if err := s.doRequest(s.tokenPriceUrl, &priceData); err != nil {
			s.logger.Errorln(err)
		}

		if priceData != nil {
			s.cachedPrice = priceData.Data.PriceUSD
		}

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
			s.mx.RLock()
			watcher, ok := s.cachedWatcher[watcherId]
			s.mx.RUnlock()
			if !ok {
				continue
			}

			if watcher != nil && watcher.Threshold != nil && watcher.TokenPrice != nil {
				percentage := (s.cachedPrice - *watcher.TokenPrice) / *watcher.TokenPrice * 100
				roundedPercentage := math.Abs((math.Round(percentage*100) / 100))
				roundedPrice := strconv.FormatFloat(s.cachedPrice, 'f', 5, 64)

				decodedPushToken, err := base64.StdEncoding.DecodeString(watcher.PushToken)
				if err != nil {
					s.logger.Errorln(err)
				}

				if percentage >= float64(*watcher.Threshold) {
					if *watcher.PriceNotification == ON {
						data := map[string]interface{}{"type": "price-alert", "percentage": roundedPercentage}
						title := "Price Alert"
						body := fmt.Sprintf("🚀 AMB Price changed on +%v%s! Current price $%v\n", roundedPercentage, "%", roundedPrice)
						sent := false

						response, err := s.cloudMessagingSvc.SendMessage(ctx, title, body, string(decodedPushToken), data)
						if err != nil {
							s.logger.Errorln(err)
						}

						if response != nil {
							sent = true
						}

						watcher.AddNotification(title, body, sent, time.Now())
					}

					watcher.SetTokenPrice(s.cachedPrice)

					if err := s.repository.UpdateWatcher(ctx, watcher); err != nil {
						s.logger.Errorln(err)
					}

					s.mx.Lock()
					s.cachedWatcher[watcherId] = watcher
					s.mx.Unlock()
				}

				if percentage <= -float64(*watcher.Threshold) {
					if *watcher.PriceNotification == ON {
						data := map[string]interface{}{"type": "price-alert", "percentage": roundedPercentage}
						title := "Price Alert"
						body := fmt.Sprintf("🔻 AMB Price changed on -%v%s! Current price $%v\n", roundedPercentage, "%", roundedPrice)
						sent := false

						response, err := s.cloudMessagingSvc.SendMessage(ctx, title, body, string(decodedPushToken), data)
						if err != nil {
							s.logger.Errorln(err)
						}

						if response != nil {
							sent = true
						}

						watcher.AddNotification(title, body, sent, time.Now())
					}

					watcher.SetTokenPrice(s.cachedPrice)

					if err := s.repository.UpdateWatcher(ctx, watcher); err != nil {
						s.logger.Errorln(err)
					}

					s.mx.Lock()
					s.cachedWatcher[watcherId] = watcher
					s.mx.Unlock()
				}

			}

			time.Sleep(330 * time.Second)
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
			s.mx.RUnlock()
			if !ok {
				continue
			}

			if watcher != nil && (watcher.Addresses != nil && len(*watcher.Addresses) > 0) {
				for _, address := range *watcher.Addresses {
					var apiAddressData *ApiAddressData

					if err := s.doRequest(fmt.Sprintf("%s/addresses/%s/all", s.explorerUrl, address.Address), &apiAddressData); err != nil {
						s.logger.Errorln(err)
					}

					if apiAddressData != nil && len(apiAddressData.Data) > 0 {

						missedTxs := []Tx{}

						for i := range apiAddressData.Data {
							if address.LastTx == nil || (*address.LastTx != apiAddressData.Data[i].Hash && (i+1) < len(apiAddressData.Data) && *address.LastTx == apiAddressData.Data[i+1].Hash) {
								missedTxs = append(missedTxs, apiAddressData.Data[i])

								decodedPushToken, err := base64.StdEncoding.DecodeString(watcher.PushToken)
								if err != nil {
									s.logger.Errorln(err)
								}

								for i := len(missedTxs) - 1; i >= 0; i-- {
									if (len(missedTxs[i].From) == 0 || missedTxs[i].From == "") || (len(missedTxs[i].To) == 0 || missedTxs[i].To == "") {
										continue
									}

									if *watcher.TxNotification == ON {
										data := map[string]interface{}{"type": "transaction-alert", "timestamp": missedTxs[i].Timestamp}

										title := "AMB-Net Tx Alert"
										cutFromAddress := fmt.Sprintf("%s...%s", missedTxs[i].From[:5], missedTxs[i].From[len(missedTxs[i].From)-5:])
										cutToAddress := fmt.Sprintf("%s...%s", missedTxs[i].To[:5], missedTxs[i].To[len(missedTxs[i].From)-5:])
										roundedAmount := strconv.FormatFloat(missedTxs[i].Value.Ether, 'f', 2, 64)

										body := fmt.Sprintf("From: %s\nTo: %s\nAmount: %s", cutFromAddress, cutToAddress, roundedAmount)
										sent := false

										response, err := s.cloudMessagingSvc.SendMessage(ctx, title, body, string(decodedPushToken), data)
										if err != nil {
											s.logger.Errorln(err)
										}

										if response != nil {
											sent = true
										}

										watcher.AddNotification(title, body, sent, time.Now())
									}

									watcher.SetLastTx(address.Address, missedTxs[i].Hash)

									if err := s.repository.UpdateWatcher(ctx, watcher); err != nil {
										s.logger.Errorln(err)
									}

									s.mx.Lock()
									s.cachedWatcher[watcherId] = watcher
									s.mx.Unlock()
								}

								break
							}

							missedTxs = append(missedTxs, apiAddressData.Data[i])

						}
					}
				}
			}

			time.Sleep(10 * time.Second)
		}
	}
}

func (s *service) GetWatcher(ctx context.Context, pushToken string) (*Watcher, error) {
	encodePushToken := base64.StdEncoding.EncodeToString([]byte(pushToken))

	watcher, err := s.repository.GetWatcher(ctx, bson.M{"push_token": encodePushToken})
	if err != nil {
		return nil, err
	}

	if watcher == nil {
		return nil, errors.New("watcher not found")
	}

	return watcher, nil
}

func (s *service) GetWatcherHistoryPrices(ctx context.Context) *CGData {
	return &CGData{Prices: s.cachedCgPrice}
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

	watcher.SetThreshold(5)
	watcher.SetTxNotification(ON)
	watcher.SetPriceNotification(ON)

	var priceData *PriceData
	if err := s.doRequest(s.tokenPriceUrl, &priceData); err != nil {
		return err
	}

	if priceData != nil {
		watcher.SetTokenPrice(priceData.Data.PriceUSD)
	}

	if err := s.repository.CreateWatcher(ctx, watcher); err != nil {
		return err
	}

	s.cachedWatcher[watcher.ID.Hex()] = watcher

	s.sentUpStopChanAndStartWatchers(ctx, watcher)

	return nil
}

func (s *service) UpdateWatcher(ctx context.Context, pushToken string, addresses *[]string, threshold *float64, txNotification, priceNotification *string) error {
	encodePushToken := base64.StdEncoding.EncodeToString([]byte(pushToken))

	watcher, err := s.repository.GetWatcher(ctx, bson.M{"push_token": encodePushToken})
	if err != nil {
		return err
	}

	if watcher == nil {
		return errors.New("watcher not found")
	}

	if addresses != nil && len(*addresses) > 0 {
		for _, address := range *addresses {

			if watcher.Addresses != nil {
				for _, v := range *watcher.Addresses {
					if address == v.Address {
						return errors.New("address already is watching")
					}
				}
			}

			watcher.AddAddress(address)

			var apiAddressData *ApiAddressData
			if err := s.doRequest(fmt.Sprintf("%s/addresses/%s/all", s.explorerUrl, address), &apiAddressData); err != nil {
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

	if txNotification != nil && *txNotification != "" {
		watcher.SetTxNotification(*txNotification)
	}

	if priceNotification != nil && *priceNotification != "" {
		watcher.SetPriceNotification(*priceNotification)
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

func (s *service) DeleteWatcherAddresses(ctx context.Context, pushToken string, addresses []string) error {
	encodePushToken := base64.StdEncoding.EncodeToString([]byte(pushToken))

	filters := bson.M{"push_token": encodePushToken}

	watcher, err := s.repository.GetWatcher(ctx, filters)
	if err != nil {
		return err
	}

	if watcher == nil {
		return errors.New("watcher not found")
	}

	for _, address := range addresses {
		watcher.DeleteAddress(address)
	}

	if err := s.repository.UpdateWatcher(ctx, watcher); err != nil {
		return err
	}

	s.mx.Lock()
	s.cachedWatcher[watcher.ID.Hex()] = watcher
	s.mx.Unlock()

	return nil
}

func (s *service) sentUpStopChanAndStartWatchers(ctx context.Context, watcher *Watcher) {
	s.mx.Lock()
	stopChan := make(chan struct{})
	s.cachedChan[watcher.ID.Hex()] = stopChan
	s.mx.Unlock()

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
