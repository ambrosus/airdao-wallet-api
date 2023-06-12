package watcher

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Address struct {
	Address string  `bson:"address"`
	LastTx  *string `bson:"last_tx"`
}

type HistoryNotification struct {
	Body      string    `bson:"body"`
	Timestamp time.Time `bson:"timestamp"`
}

type Watcher struct {
	ID primitive.ObjectID `bson:"_id"`

	PushToken string `bson:"push_token"`
	Threshold *int   `bson:"threshold"`

	Addresses *[]*Address `bson:"addresses"`

	HistoryNotifications *[]*HistoryNotification `bson:"history_notifications"`

	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

func NewWatcher(pushToken string) (*Watcher, error) {
	if pushToken == "" {
		return nil, errors.New("invalid push token")
	}

	return &Watcher{
		ID: primitive.NewObjectID(),

		PushToken: pushToken,
		Threshold: nil,

		Addresses:            nil,
		HistoryNotifications: nil,

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
	for i, v := range *w.Addresses {
		if v.Address == address {
			*w.Addresses = append((*w.Addresses)[:i], (*w.Addresses)[i+1:]...)
			break
		}
	}

	w.UpdatedAt = time.Now()
}

func (w *Watcher) SetLastTx(address string, tx string) {
	for _, v := range *w.Addresses {
		if v.Address == address {
			v.LastTx = &tx
		}
	}

	w.UpdatedAt = time.Now()
}

func (w *Watcher) SetThreshold(threshold int) {
	w.Threshold = &threshold
	w.UpdatedAt = time.Now()
}
