package pkg

import (
	"go.mongodb.org/mongo-driver/bson"
)

type Item = bson.M
type Query = bson.D

type Resource = []byte
type Resources map[string]Resource

type DB interface {
	Put(*Item, ...PutOpt) error
	Get([]byte) (*Item, error)
	GetResources([]byte) (Resources, error)
	Find(Query) (Iterator, error)
	Delete(Query) error
	Compact() error
	Close() error
}

type Iterator interface {
	Next() bool
	Key() ([]byte, error)
	Value() (*Item, error)
	Err() error
	Release() error
}

type PutOpt func(o *putOpt)
type putOpt struct {
	resources Resources
	key       []byte
}

func WithResources(rc Resources) PutOpt {
	return func(o *putOpt) {
		o.resources = rc
	}
}

func WithKey(key []byte) PutOpt {
	return func(o *putOpt) {
		o.key = key
	}
}

func applyPutOpts(os []PutOpt) putOpt {
	rc := putOpt{}
	for _, o := range os {
		o(&rc)
	}
	return rc
}
