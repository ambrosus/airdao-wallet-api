package watcher

import (
	"bytes"
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

	TransactionWatch(ctx context.Context, address string, txHash string, cache map[string]bool)
	ApiPriceWatch(ctx context.Context)
	PriceWatch(ctx context.Context, watcherId string, stopChan chan struct{})
	CGWatch(ctx context.Context)

	GetExplorerId() string

	GetWatcher(ctx context.Context, pushToken string) (*Watcher, error)
	GetWatcherHistoryPrices(ctx context.Context) *CGData
	CreateWatcher(ctx context.Context, pushToken string) error
	UpdateWatcher(ctx context.Context, pushToken string, addresses *[]string, threshold *float64, txNotification, priceNotification *string) error
	DeleteWatcher(ctx context.Context, pushToken string) error
	DeleteWatcherAddresses(ctx context.Context, pushToken string, addresses []string) error
}

type watchers struct {
	watchers    map[string]*Watcher
}

type service struct {
	repository        Repository
	cloudMessagingSvc cloudmessaging.Service
	logger            *zap.SugaredLogger

	explorerUrl   string
	tokenPriceUrl string
	callbackUrl   string
	explorerToken string

	mx                     sync.RWMutex
	cachedWatcher          map[string]*Watcher
	cachedWatcherByAddress map[string]*watchers
	cachedChan             map[string]chan struct{}
	cachedPrice            float64
	cachedCgPrice          [][]float64
}

func NewService(
	repository Repository,
	cloudMessagingSvc cloudmessaging.Service,
	logger *zap.SugaredLogger,
	explorerUrl string,
	tokenPriceUrl string,
	callbackUrl string,
	explorerToken string,
) (Service, error) {
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
		callbackUrl: callbackUrl,
		explorerToken: explorerToken,

		cachedChan:             make(map[string]chan struct{}),
		cachedWatcher:          make(map[string]*Watcher),
		cachedWatcherByAddress: make(map[string]*watchers),
		cachedPrice:            0,
		cachedCgPrice:          [][]float64{},
	}, nil
}

func (self *watchers) Add(watcher *Watcher) {
	if self.watchers == nil {
		self.watchers = make(map[string]*Watcher)
	}
	self.watchers[watcher.PushToken] = watcher
}

func (self *watchers) Remove(pushToken string) {
	if self.watchers != nil {
		delete(self.watchers, pushToken)
	}
}

func (self *watchers) IsEmpty() bool {
    return self.watchers == nil || len(self.watchers) == 0
}

func (s *service) Init(ctx context.Context) error {
	var priceData *PriceData
	if err := s.doRequest(s.tokenPriceUrl, nil, &priceData); err != nil {
		return err
	}

	if priceData != nil {
		s.cachedPrice = priceData.Data.PriceUSD
	}

	go s.CGWatch(ctx)
	go s.ApiPriceWatch(ctx)
	go s.keepAlive(ctx)

	return nil
}

func (s *service) keepAlive(ctx context.Context) {
	loadWatchers := true
	for {
		var req bytes.Buffer
		req.WriteString("{\"id\":\"")
		req.WriteString(s.explorerToken)
		req.WriteString("\",\"action\":\"init\",\"url\":\"")
		req.WriteString(s.callbackUrl)
		req.WriteString("\"}")
		if err := s.doRequest(fmt.Sprintf("%s/watch", s.explorerUrl), &req, nil); err != nil {
			s.logger.Errorln(err)
			time.Sleep(5 * time.Second)
			continue
		}

		if loadWatchers {
			loadWatchers = false
			page := 1
			for {
				watchers, err := s.repository.GetWatcherList(ctx, bson.M{}, page)
				if err != nil {
					s.logger.Errorf("keepAlive repository.GetWatcherList error %v", err)
					break
				}

				if watchers == nil {
			    		break
				}

				for _, watcher := range watchers {
					s.mx.Lock()
					s.cachedWatcher[watcher.PushToken] = watcher
					s.mx.Unlock()

					s.setUpStopChanAndStartWatchers(ctx, watcher)
				}

				page++
			}
		} else {
			watchers := make([]*Watcher, 0)
			s.mx.Lock()
			for _, watcher := range s.cachedWatcher {
				watchers = append(watchers, watcher)
			}
			s.mx.Unlock()
			for _, watcher := range watchers {
				if watcher.Addresses != nil && len(*watcher.Addresses) > 0 {
					var req bytes.Buffer
					req.WriteString("{\"id\":\"")
					req.WriteString(s.explorerToken)
					req.WriteString("\",\"action\":\"subscribe\",\"addresses\":[")
					for i, address := range *watcher.Addresses {
						if i != 0 {
							req.WriteString(",\"")
						} else {
							req.WriteString("\"")
						}
						req.WriteString(address.Address)
						req.WriteString("\"")
					}
					req.WriteString("]}")
					if err := s.doRequest(fmt.Sprintf("%s/watch", s.explorerUrl), &req, nil); err != nil {
						s.logger.Errorln(err)
					}
				}
			}
		}

		tries := 6
		for {
			req.Reset()
			req.WriteString("{\"id\":\"")
			req.WriteString(s.explorerToken)
			req.WriteString("\",\"action\":\"check\"}")
			if err := s.doRequest(fmt.Sprintf("%s/watch", s.explorerUrl), &req, nil); err != nil {
				s.logger.Errorln(err)
				if tries != 0 {
					tries--
					time.Sleep(5 * time.Second)
					continue
				}
				break;
			}
			time.Sleep(30 * time.Second)
			tries = 6
		}
	}
}

func (s *service) CGWatch(ctx context.Context) {
	for {
		var cgData *CGData
		if err := s.doRequest("https://api.coingecko.com/api/v3/coins/amber/market_chart?vs_currency=usd&days=30", nil, &cgData); err != nil {
			s.logger.Errorln(err)
		}

		s.cachedCgPrice = cgData.Prices

		time.Sleep(12 * time.Hour)
	}
}

func (s *service) ApiPriceWatch(ctx context.Context) {
	for {
		var priceData *PriceData
		if err := s.doRequest(s.tokenPriceUrl, nil, &priceData); err != nil {
			s.logger.Errorln(err)
		}

		if priceData != nil {
			s.cachedPrice = priceData.Data.PriceUSD
		}

		time.Sleep(5 * time.Minute)
	}
}

func (s *service) PriceWatch(ctx context.Context, watcherId string, stopChan chan struct{}) {
	s.mx.RLock()
	watcher, ok := s.cachedWatcher[watcherId]
	s.mx.RUnlock()
	if !ok || watcher == nil {
		return
	}

	decodedPushToken, err := base64.StdEncoding.DecodeString(watcher.PushToken)
	if err != nil {
		s.logger.Errorf("PriceWatch base64.StdEncoding.DecodeString error %v\n", err)
		return
	}

	for {
		select {
		case <-stopChan:
			// fmt.Println("Stopping price goroutine...")
			return
		default:
			if watcher.Threshold != nil && watcher.TokenPrice != nil {
				percentage := (s.cachedPrice - *watcher.TokenPrice) / *watcher.TokenPrice * 100
				roundedPercentage := math.Abs((math.Round(percentage*100) / 100))
				roundedPrice := strconv.FormatFloat(s.cachedPrice, 'f', 5, 64)

				if percentage >= float64(*watcher.Threshold) {
					if watcher.PriceNotification == ON {
						data := map[string]interface{}{"type": "price-alert", "percentage": roundedPercentage}
						title := "Price Alert"
						body := fmt.Sprintf("🚀 AMB Price changed on +%v%s! Current price $%v\n", roundedPercentage, "%", roundedPrice)
						sent := false

						response, err := s.cloudMessagingSvc.SendMessage(ctx, title, body, string(decodedPushToken), data)
						if err != nil {
							s.logger.Errorf("PriceWatch (Up) cloudMessagingSvc.SendMessage error %v\n", err)
							if err.Error() == "http error status: 404; reason: app instance has been unregistered; code: registration-token-not-registered; details: Requested entity was not found." {
								return
							}
						}

						if response != nil {
							sent = true
						}

						watcher.AddNotification(title, body, sent, time.Now())
					}

					watcher.SetTokenPrice(s.cachedPrice)

					if err := s.repository.UpdateWatcher(ctx, watcher); err != nil {
						s.logger.Errorf("PriceWatch (Up) repository.UpdateWatcher error %v\n", err)
					}
				}

				if percentage <= -float64(*watcher.Threshold) {
					if watcher.PriceNotification == ON {
						data := map[string]interface{}{"type": "price-alert", "percentage": roundedPercentage}
						title := "Price Alert"
						body := fmt.Sprintf("🔻 AMB Price changed on -%v%s! Current price $%v\n", roundedPercentage, "%", roundedPrice)
						sent := false

						response, err := s.cloudMessagingSvc.SendMessage(ctx, title, body, string(decodedPushToken), data)
						if err != nil {
							s.logger.Errorf("PriceWatch (Down) cloudMessagingSvc.SendMessage error %v\n", err)
							if err.Error() == "http error status: 404; reason: app instance has been unregistered; code: registration-token-not-registered; details: Requested entity was not found." {
								return
							}
						}

						if response != nil {
							sent = true
						}

						watcher.AddNotification(title, body, sent, time.Now())
					}

					watcher.SetTokenPrice(s.cachedPrice)

					if err := s.repository.UpdateWatcher(ctx, watcher); err != nil {
						s.logger.Errorf("PriceWatch (Down) repository.UpdateWatcher error %v\n", err)
					}
				}

			}

			time.Sleep(330 * time.Second)
		}
	}
}

func (s *service) TransactionWatch(ctx context.Context, address string, txHash string, cache map[string]bool) {
	s.mx.RLock()
	watchers, ok := s.cachedWatcherByAddress[address]
	s.mx.RUnlock()
	if !ok {
		return
	}

	var cutFromAddress string
	var cutToAddress string
	var roundedAmount string
	var tokenSymbol string
	var data map[string]interface{}
	takeTx := true

	for _, watcher := range watchers.watchers {
		if watcher != nil && watcher.TxNotification == ON && (watcher.Addresses != nil && len(*watcher.Addresses) > 0) {
			itemId := txHash + watcher.PushToken
			if _, ok := cache[itemId]; !ok {
				if takeTx {
					takeTx = false
					var apiTxData *ApiTxData
					if err := s.doRequest(fmt.Sprintf("%s/transactions/%s", s.explorerUrl, txHash), nil, &apiTxData); err != nil {
						s.logger.Errorf("TransactionWatch doRequest error %v\n", err)
						return
					}
					if apiTxData == nil || len(apiTxData.Data) == 0 {
						s.logger.Errorln("TransactionWatch empty tx response")
						return
					}
					tx := &apiTxData.Data[0]
					if tx.Value.Ether == 0 {
						return
					}
					if len(tx.From) > 0 && tx.From != "" {
						cutFromAddress = fmt.Sprintf("%s...%s", tx.From[:5], tx.From[len(tx.From)-5:])
					}
					if len(tx.To) > 0 && tx.To != "" {
						cutToAddress = fmt.Sprintf("%s...%s", tx.To[:5], tx.To[len(tx.To)-5:])
					}
					roundedAmount = strconv.FormatFloat(tx.Value.Ether, 'f', 2, 64)
					if tx.Value.Symbol == nil {
						tokenSymbol = "AMB"
					} else {
						if *tx.Value.Symbol == "" {
							tokenSymbol = "HPT"
						} else {
							tokenSymbol = *tx.Value.Symbol
						}
					}
					data = map[string]interface{}{"type": "transaction-alert", "timestamp": tx.Timestamp, "sender": cutFromAddress, "to": cutToAddress}
				}

				decodedPushToken, err := base64.StdEncoding.DecodeString(watcher.PushToken)
				if err != nil {
					s.logger.Errorf("TransactionWatch base64.StdEncoding.DecodeString error %v\n", err)
					continue
				}

				title := "AMB-Net Tx Alert"
				body := fmt.Sprintf("From: %s\nTo: %s\nAmount: %s %s", cutFromAddress, cutToAddress, roundedAmount, tokenSymbol)
				sent := false

				response, err := s.cloudMessagingSvc.SendMessage(ctx, title, body, string(decodedPushToken), data)
				if err != nil {
					s.logger.Errorf("TransactionWatch cloudMessagingSvc.SendMessage error %v\n", err)
					if err.Error() == "http error status: 404; reason: app instance has been unregistered; code: registration-token-not-registered; details: Requested entity was not found." {
						s.mx.RLock()
						watchers.Remove(watcher.PushToken)
						s.mx.RUnlock()
					}
				}

				if response != nil {
					sent = true
				}

				watcher.AddNotification(title, body, sent, time.Now())

				fmt.Printf("Tx notify: %v:%v\n", txHash, watcher.PushToken)
				cache[itemId] = true
			}

			watcher.SetLastTx(address, txHash)

			if err := s.repository.UpdateWatcher(ctx, watcher); err != nil {
				s.logger.Errorf("TransactionWatch repository.UpdateWatcher error %v\n", err)
			}
		}
	}
}

func (s *service) GetWatcher(ctx context.Context, pushToken string) (*Watcher, error) {
	encodePushToken := base64.StdEncoding.EncodeToString([]byte(pushToken))
	var watcher *Watcher
	var ok bool

	s.mx.Lock()
	watcher, ok = s.cachedWatcher[encodePushToken]
	s.mx.Unlock()

	if ok && watcher != nil {
		return watcher, nil
	}

	watcher, err := s.repository.GetWatcher(ctx, bson.M{"push_token": encodePushToken})
	if err != nil {
		return nil, err
	}

	if watcher == nil {
		return nil, errors.New("watcher not found")
	}

	s.mx.Lock()
	s.cachedWatcher[encodePushToken] = watcher
	s.mx.Unlock()

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
	if err := s.doRequest(s.tokenPriceUrl, nil, &priceData); err != nil {
		return err
	}

	if priceData != nil {
		watcher.SetTokenPrice(priceData.Data.PriceUSD)
	}

	if err := s.repository.CreateWatcher(ctx, watcher); err != nil {
		return err
	}

	s.mx.Lock()
	s.cachedWatcher[watcher.PushToken] = watcher
	s.mx.Unlock()

	s.setUpStopChanAndStartWatchers(ctx, watcher)

	return nil
}

func (s *service) UpdateWatcher(ctx context.Context, pushToken string, addresses *[]string, threshold *float64, txNotification, priceNotification *string) error {
	watcher, err := s.GetWatcher(ctx, pushToken)
	if err != nil {
		return err
	}
	if watcher == nil {
		return errors.New("watcher not found")
	}

	if addresses != nil && len(*addresses) > 0 {
		var req bytes.Buffer
		req.WriteString("{\"id\":\"")
		req.WriteString(s.explorerToken)
		req.WriteString("\",\"action\":\"subscribe\",\"addresses\":[")
		for i, address := range *addresses {
			if watcher.Addresses != nil {
				for _, v := range *watcher.Addresses {
					if address == v.Address {
						return errors.New("address already is watching")
					}
				}
			}

			if i != 0 {
				req.WriteString(",\"")
			} else {
				req.WriteString("\"")
			}
			req.WriteString(address)
			req.WriteString("\"")

			watcher.AddAddress(address)
			s.addWatcherForAddress(address, watcher)
		}
		req.WriteString("]}")
		if err := s.doRequest(fmt.Sprintf("%s/watch", s.explorerUrl), &req, nil); err != nil {
			s.logger.Errorln(err)
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

	return nil
}

func (s *service) DeleteWatcher(ctx context.Context, pushToken string) error {
	encodePushToken := base64.StdEncoding.EncodeToString([]byte(pushToken))

	watcher, err := s.GetWatcher(ctx, pushToken)
	if err != nil {
		return err
	}
	if watcher == nil {
		return errors.New("watcher not found")
	}

	if err := s.repository.DeleteWatcher(ctx, bson.M{"push_token": encodePushToken}); err != nil {
		return err
	}

	s.mx.RLock()
	watcherStopChan := s.cachedChan[watcher.PushToken]
	s.mx.RUnlock()

	close(watcherStopChan)
	s.mx.RLock()
	delete(s.cachedWatcher, watcher.PushToken)
	delete(s.cachedChan, watcher.PushToken)
	s.mx.RUnlock()

	if watcher.Addresses != nil && len(*watcher.Addresses) > 0 {
		var req bytes.Buffer
		req.WriteString("{\"id\":\"")
		req.WriteString(s.explorerToken)
		req.WriteString("\",\"action\":\"unsubscribe\",\"addresses\":[")
		first := true
		empty := true
		for _, address := range *watcher.Addresses {
			remove := true
			s.mx.RLock()
			if watchers, ok := s.cachedWatcherByAddress[address.Address]; ok {
				watchers.Remove(watcher.PushToken)
				remove = watchers.IsEmpty()
			}
			s.mx.RUnlock()
			if remove {
				empty = false
				if first {
					first = false
					req.WriteString("\"")
				} else {
					req.WriteString(",\"")
				}
				req.WriteString(address.Address)
				req.WriteString("\"")
			}
		}
		if !empty {
			req.WriteString("]}")
			if err := s.doRequest(fmt.Sprintf("%s/watch", s.explorerUrl), &req, nil); err != nil {
				s.logger.Errorln(err)
			}
		}
	}

	return nil
}

func (s *service) DeleteWatcherAddresses(ctx context.Context, pushToken string, addresses []string) error {
	watcher, err := s.GetWatcher(ctx, pushToken)
	if err != nil {
		return err
	}
	if watcher == nil {
		return errors.New("watcher not found")
	}

	var req bytes.Buffer
	req.WriteString("{\"id\":\"")
	req.WriteString(s.explorerToken)
	req.WriteString("\",\"action\":\"unsubscribe\",\"addresses\":[")
	first := true
	empty := true
	for _, address := range addresses {
		remove := true
		watcher.DeleteAddress(address)
		s.mx.RLock()
		if watchers, ok := s.cachedWatcherByAddress[address]; ok {
			watchers.Remove(watcher.PushToken)
			remove = watchers.IsEmpty()
		}
		s.mx.RUnlock()
		if remove {
			empty = false
			if first {
				first = false
				req.WriteString("\"")
			} else {
				req.WriteString(",\"")
			}
			req.WriteString(address)
			req.WriteString("\"")
		}
	}
	if !empty {
		req.WriteString("]}")
		if err := s.doRequest(fmt.Sprintf("%s/watch", s.explorerUrl), &req, nil); err != nil {
			s.logger.Errorln(err)
		}
	}

	if err := s.repository.UpdateWatcher(ctx, watcher); err != nil {
		return err
	}

	return nil
}

func (s *service) setUpStopChanAndStartWatchers(ctx context.Context, watcher *Watcher) {
	s.mx.Lock()
	stopChan := make(chan struct{})
	s.cachedChan[watcher.PushToken] = stopChan
	s.mx.Unlock()

	if watcher.Addresses != nil && len(*watcher.Addresses) > 0 {
		var req bytes.Buffer
		req.WriteString("{\"id\":\"")
		req.WriteString(s.explorerToken)
		req.WriteString("\",\"action\":\"subscribe\",\"addresses\":[")
		for i, address := range *watcher.Addresses {
			if i != 0 {
				req.WriteString(",\"")
			} else {
				req.WriteString("\"")
			}
			req.WriteString(address.Address)
			req.WriteString("\"")

			s.addWatcherForAddress(address.Address, watcher)
		}
		req.WriteString("]}")
		if err := s.doRequest(fmt.Sprintf("%s/watch", s.explorerUrl), &req, nil); err != nil {
			s.logger.Errorln(err)
		}
	}

	go s.PriceWatch(ctx, watcher.PushToken, stopChan)
}

func (s *service) addWatcherForAddress(address string, watcher *Watcher) {
	var items *watchers
	var ok bool
	s.mx.Lock()
	items, ok = s.cachedWatcherByAddress[address]
	if !ok || items == nil {
		items = new(watchers)
		s.cachedWatcherByAddress[address] = items
	}
	items.Add(watcher)
	s.mx.Unlock()
}

func (s *service) doRequest(url string, body io.Reader, res interface{}) error {
	client := &http.Client{}
	var method string

	if body != nil {
		method = "POST"
	} else {
		method = "GET"
	}

	fmt.Printf("Request %v %v\n", method, url)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}

	if body != nil {
		if reader, err := req.GetBody(); err == nil {
			if reqBody, err := io.ReadAll(reader); err == nil {
				fmt.Printf("Body: %v\n", string(reqBody))
			}
		}
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("Not found")
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("Response: %v\n", string(respBody))

	if res != nil {
		if err := json.Unmarshal(respBody, res); err != nil {
			return err
		}
	}

	return nil
}

func (self *service) GetExplorerId() string {
    return self.explorerToken
}
