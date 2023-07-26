package watcher

import (
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Address struct {
	Address string  `json:"address" bson:"address"`
	LastTx  *string `json:"last_tx" bson:"last_tx"`
}

type HistoryNotification struct {
	Title     string    `json:"title" bson:"title"`
	Body      string    `json:"body" bson:"body"`
	Sent      bool      `json:"sent" bson:"sent"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}

type Watcher struct {
	ID primitive.ObjectID `json:"id" bson:"_id"`

	PushToken         string   `json:"push_token" bson:"push_token"`
	Threshold         *float64 `json:"threshold" bson:"threshold"`
	TokenPrice        *float64 `json:"token_price" bson:"token_price"`
	TxNotification    string   `json:"tx_notification" bson:"tx_notification"`
	PriceNotification string   `json:"price_notification" bson:"price_notification"`

	Addresses *[]*Address `json:"addresses" bson:"addresses"`

	HistoricalNotifications *[]*HistoryNotification `json:"historical_notifications" bson:"historical_notifications"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

func NewWatcher(pushToken string) (*Watcher, error) {
	if pushToken == "" {
		return nil, errors.New("invalid push token")
	}

	return &Watcher{
		ID: primitive.NewObjectID(),

		PushToken:         pushToken,
		Threshold:         nil,
		TokenPrice:        nil,
		TxNotification:    "",
		PriceNotification: "",

		Addresses:               nil,
		HistoricalNotifications: nil,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (w *Watcher) AddAddress(address string) {
	if w.Addresses == nil {
		w.Addresses = &[]*Address{{
			Address: address,
		}}
	} else {
		*w.Addresses = append((*w.Addresses), &Address{Address: address, LastTx: nil})
	}
	w.UpdatedAt = time.Now()
}

func (w *Watcher) DeleteAddress(address string) {
	if w.Addresses != nil {
		for i, v := range *w.Addresses {
			if v.Address == address {
				*w.Addresses = append((*w.Addresses)[:i], (*w.Addresses)[i+1:]...)
				w.UpdatedAt = time.Now()
				break
			}
		}
	}
}

func (w *Watcher) SetLastTx(address string, tx string) {
	for _, v := range *w.Addresses {
		if v.Address == address {
			v.LastTx = &tx
		}
	}

	w.UpdatedAt = time.Now()
}

func (w *Watcher) SetThreshold(threshold float64) {
	fmt.Printf("SetThreshold %v\n", threshold)
	w.Threshold = &threshold
	w.UpdatedAt = time.Now()
}

func (w *Watcher) SetTokenPrice(price float64) {
	fmt.Printf("SetTokenPrice %v\n", price)
	w.TokenPrice = &price
	w.UpdatedAt = time.Now()
}

func (w *Watcher) AddNotification(title, body string, sent bool, timestamp time.Time) {
	if w.HistoricalNotifications == nil {
		w.HistoricalNotifications = &[]*HistoryNotification{{
			Title:     title,
			Body:      body,
			Sent:      sent,
			Timestamp: timestamp,
		}}
	} else {
		*w.HistoricalNotifications = append((*w.HistoricalNotifications), &HistoryNotification{
			Title: title, Body: body, Sent: sent, Timestamp: timestamp,
		})
	}
	w.UpdatedAt = time.Now()
}

func (w *Watcher) SetTxNotification(v string) {
	w.TxNotification = v
	w.UpdatedAt = time.Now()
}

func (w *Watcher) SetPriceNotification(v string) {
	w.PriceNotification = v
	w.UpdatedAt = time.Now()
}
