package pkg

import (
	"github.com/dgraph-io/badger/v3"
	"go.mongodb.org/mongo-driver/bson"
)

type Item = bson.M
type Query = bson.D

type Resource = []byte
type Resources map[string]Resource

var (
	ErrKeyNotFound = badger.ErrKeyNotFound
)

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
	// ListBucket gets all user buckets
	ListBucket() ([]string, error)
}

type Bucket interface {
	// Put saves val by key
	Put(key, val []byte, opts ...PutOption) error
	// PutVal saves value and return auto inc id (binary with big endian)
	// Do not use it with Put func in the same bucket
	// Items may be overwritten with auto generated key
	PutVal(val []byte, opts ...PutOption) ([]byte, error)
	// Get gets val by key
	Get(key []byte) ([]byte, *Meta, error)
	// GetAt gets val from offset, it returns num of bytes
	// It returns io.EOF if no bytes were read
	GetAt(key, buf []byte, offset int) (int, error)
	// GetMeta returns meta info
	// It returns nil, nil when there is no meta
	GetMeta(key []byte) (*Meta, error)
	// Exists check if key exists
	Exists(key []byte) (bool, error)
	// Delete removes val by key
	Delete(key []byte) error
	// Range returns iterator for [beginKey, endKey), all for nil, nil
	Range(beginKey, endKey []byte, reverse bool) (Iterator, error)
	// Count returns item count of [beginKey, endKey), all for nil, nil
	Count(beginKey, endKey []byte) (int, error)
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
