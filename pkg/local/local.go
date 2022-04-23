package local

import (
	"encoding/binary"
	"time"

	"github.com/dgraph-io/badger/v3"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/jialeicui/archivedb/pkg"
)

const (
	mainBucket = "main"
	ossBucket  = "oss"
)

type kv struct {
	store     *badger.DB
	main, oss *bucket
}

func (d *kv) Put(item *pkg.Item) error {
	content, err := bson.Marshal(item)
	if err != nil {
		return err
	}
	tm := time.Now().Unix()
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, uint64(tm))

	return d.main.put(key, content)
}

func (d *kv) Find(query pkg.Query) (pkg.Iterator, error) {
	return d.main.find(query)
}

func (d kv) Delete(query pkg.Query) error {
	return nil
}

func (d kv) Compact() error {
	return d.store.Flatten(1)
}

func (d kv) Close() error {
	return d.store.Close()
}

func New(path string) (pkg.DB, error) {
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
	query  pkg.Query
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

func (i *iterator) Value() (*pkg.Item, error) {
	var item = new(pkg.Item)
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

func (b *bucket) find(query pkg.Query) (pkg.Iterator, error) {
	txn := b.store.NewTransaction(false)
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	return &iterator{
		query:  query,
		txn:    txn,
		iter:   iter,
		prefix: b.prefix,
	}, nil
}
