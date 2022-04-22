package pkg

import (
	"go.mongodb.org/mongo-driver/bson"
)

type Item = bson.M
type Query = bson.D

type DB interface {
	Put(*Item) error
	Find(Query) (Iterator, error)
	Delete(Query) error
	Compact() error

	Close() error
}

type Iterator interface {
	Next() bool
	Value() (*Item, error)
	Err() error

	Release() error
}
