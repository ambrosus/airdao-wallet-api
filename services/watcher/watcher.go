package watcher

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Watcher struct {
	ID        primitive.ObjectID `bson:"_id"`
	Address   string             `bson:"address"`
	PushToken string             `bson:"push_token"`

	LastTx *string `bson:"last_tx"`

	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

func NewWatcher(address, pushToken string) (*Watcher, error) {
	if address == "" {
		return nil, errors.New("invalid address")
	}
	if pushToken == "" {
		return nil, errors.New("invalid push token")
	}

	return &Watcher{
		ID:        primitive.NewObjectID(),
		Address:   address,
		PushToken: pushToken,

		LastTx: nil,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (w *Watcher) SetAddress(address string) {
	w.Address = address
	w.UpdatedAt = time.Now()
}

func (w *Watcher) SetPushToken(pushToken string) {
	w.PushToken = pushToken
	w.UpdatedAt = time.Now()
}

func (w *Watcher) SetLastTx(tx string) {
	w.LastTx = &tx
	w.UpdatedAt = time.Now()
}
