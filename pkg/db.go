package pkg

import (
	"go.mongodb.org/mongo-driver/bson"
)

type Item = bson.M
type Query = bson.D

type Resource = []byte
type Resources map[string]Resource

type DB interface {
	// CreateNamespace creates or gets namespace
	CreateNamespace(name []byte) (Namespace, error)
	// DeleteNamespace deletes namespace by name and all data in the namespace
	DeleteNamespace(name []byte) error
	// Compact do flush and compaction on db
	Compact() error
	// Close release db lock
	Close() error
}

type Namespace interface {
	// DocBucket returns builtin bucket for saving main docs
	DocBucket() DocBucket
	// ObjectBucket returns builtin bucket for saving binary resources
	ObjectBucket() Bucket
	// CreateBucket creates or gets bucket
	CreateBucket(name []byte) (Bucket, error)
	// DeleteBucket deletes bucket by name and all data in the bucket
	DeleteBucket(name []byte) error
}

type Bucket interface {
	Put(key, val []byte) error
	Get(key []byte) ([]byte, error)
	Exists(key []byte) (bool, error)
	Delete(key []byte) error
	// Range returns iterator for [beginKey, endKey), all for nil, nil
	Range(beginKey, endKey []byte) (Iterator, error)
}

type DocBucket interface {
	Bucket
	PutDoc(key []byte, val Item) error
	GetDoc(key []byte) (Item, error)
	Find(Query) (DocIterator, error)
}

type Iterator interface {
	Next() bool
	Key() ([]byte, error)
	Value() ([]byte, error)
	Err() error
	Release() error
}

type DocIterator interface {
	Iterator
	ValueDoc() (Item, error)
}
