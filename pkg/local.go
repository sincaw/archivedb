package pkg

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/v3"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	mainBucket = "main"
	ossBucket  = "oss"
)

type kv struct {
	store     *badger.DB
	main, oss *bucket

	keyCounter uint32
}

func (d *kv) generateKey() []byte {
	// 0-7: timestamp, 8-11 counter
	key := make([]byte, 12)
	tm := time.Now().Unix()
	binary.BigEndian.PutUint64(key, uint64(tm))

	// TODO make key asc per ms
	c := atomic.AddUint32(&d.keyCounter, 1)
	binary.BigEndian.PutUint16(key[8:], uint16(c%math.MaxUint16))

	return key
}

func (d *kv) Put(item *Item, opt ...PutOpt) error {
	o := applyPutOpts(opt)
	content, err := bson.Marshal(item)
	if err != nil {
		return err
	}
	key := o.key
	if key == nil {
		key = d.generateKey()
	}

	if o.resources != nil {
		for k, v := range o.resources {
			ossKey := append(key, []byte(k)...)
			err := d.oss.put(ossKey, v)
			if err != nil {
				return err
			}
		}
	}

	return d.main.put(key, content)
}

func (d *kv) Get(key []byte) (*Item, error) {
	val, err := d.main.get(key)
	if err != nil {
		return nil, err
	}
	var item = new(Item)
	err = bson.Unmarshal(val, item)
	return item, err
}

func (d *kv) GetResource(prefix []byte, name string) (Resource, error) {
	key := append(prefix, []byte(name)...)
	rc, err := d.oss.get(key)
	if err != nil {
		return nil, fmt.Errorf("get from db: %v, key %q", err, string(key))
	}
	return rc, nil
}

// TODO use bucket internal func
func (d *kv) GetResources(prefix []byte) (Resources, error) {
	var (
		rc   = make(map[string]Resource)
		txn  = d.store.NewTransaction(false)
		iter = txn.NewIterator(badger.DefaultIteratorOptions)
	)
	defer txn.Discard()
	defer iter.Close()
	prefix = append(d.oss.prefix, prefix...)
	for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
		var (
			item   = iter.Item()
			orgKey = item.Key()
			key    = orgKey[len(prefix):]
		)
		val, err := item.ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		rc[string(key)] = val
	}
	return rc, nil
}

func (d *kv) Find(query Query) (Iterator, error) {
	return d.main.find(query)
}

func (d kv) Delete(query Query) error {
	return nil
}

func (d kv) Compact() error {
	return d.store.Flatten(1)
}

func (d kv) Close() error {
	return d.store.Close()
}

func New(path string) (DB, error) {
	opt := badger.DefaultOptions(path)
	opt.Logger = nil
	d, err := badger.Open(opt)
	if err != nil {
		return nil, err
	}

	return &kv{
		store: d,
		main: &bucket{
			prefix: append([]byte(mainBucket), '-'),
			store:  d,
		},
		oss: &bucket{
			prefix: append([]byte(ossBucket), '-'),
			store:  d,
		},
	}, nil
}

type iterator struct {
	query  Query
	init   bool
	prefix []byte

	iter *badger.Iterator
	txn  *badger.Txn
}

func (i *iterator) Next() bool {
	if !i.init {
		i.iter.Seek(i.prefix)
		i.init = true
	} else {
		i.iter.Next()
	}
	return i.iter.ValidForPrefix(i.prefix)
}

func (i *iterator) Key() ([]byte, error) {
	return i.iter.Item().Key(), nil
}

func (i *iterator) Value() (*Item, error) {
	var item = new(Item)
	err := i.iter.Item().Value(func(val []byte) error {
		return bson.Unmarshal(val, item)
	})
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (i *iterator) Err() error {
	return nil
}

func (i *iterator) Release() error {
	i.iter.Close()
	i.txn.Discard()
	return nil
}

type bucket struct {
	prefix []byte
	store  *badger.DB
}

func (b bucket) key(key []byte) []byte {
	return append(b.prefix, key...)
}

func (b *bucket) put(key, val []byte) error {
	return b.store.Update(func(txn *badger.Txn) error {
		return txn.Set(b.key(key), val)
	})
}

func (b *bucket) get(key []byte) (val []byte, err error) {
	err = b.store.View(func(txn *badger.Txn) error {
		item, err := txn.Get(b.key(key))
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})
	return
}

func (b *bucket) find(query Query) (Iterator, error) {
	txn := b.store.NewTransaction(false)
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	return &iterator{
		query:  query,
		txn:    txn,
		iter:   iter,
		prefix: b.prefix,
	}, nil
}
