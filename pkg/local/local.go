package local

import (
	"encoding/binary"
	"time"

	"github.com/dgraph-io/badger/v3"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/jialeicui/archivedb/pkg"
)

type db struct {
	store *badger.DB
}

func (d *db) Put(item *pkg.Item) error {
	content, err := bson.Marshal(item)
	if err != nil {
		return err
	}
	tm := time.Now().Unix()
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, uint64(tm))
	err = d.store.Update(func(txn *badger.Txn) error {
		return txn.Set(key, content)
	})

	return err
}

func (d *db) Find(query pkg.Query) (pkg.Iterator, error) {
	txn := d.store.NewTransaction(false)
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	return &iterator{
		db:    d,
		query: query,
		txn:   txn,
		iter:  iter,
	}, nil
}

func (d db) Delete(query pkg.Query) error {
	return nil
}

func (d db) Compact() error {
	return d.store.Flatten(1)
}

func (d db) Close() error {
	return d.store.Close()
}

func New(path string) (pkg.DB, error) {
	opt := badger.DefaultOptions(path)
	opt.Logger = nil
	d, err := badger.Open(opt)
	if err != nil {
		return nil, err
	}

	return &db{
		store: d,
	}, nil
}

type iterator struct {
	db    *db
	query pkg.Query
	init  bool

	iter *badger.Iterator
	txn  *badger.Txn
}

func (i *iterator) Next() bool {
	if !i.init {
		i.iter.Rewind()
		i.init = true
	} else {
		i.iter.Next()
	}
	return i.iter.Valid()
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
