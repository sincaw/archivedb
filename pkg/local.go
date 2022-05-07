package pkg

import (
	"encoding/json"
	"github.com/dgraph-io/badger/v3"
	"go.mongodb.org/mongo-driver/bson"
	"sync"
)

const (
	dbDataPrefix = "d"
	dbMetaPrefix = "m"

	builtinBucketPrefix = "b"
	otherBucketPrefix   = "o"

	builtinDocBucketName    = "d"
	builtinObjectBucketName = "o"
)

type kv struct {
	sync.Mutex
	store      *badger.DB
	namespaces map[string]*ns
}

type kvMeta struct {
	Namespaces []string `json:"namespaces"`
}

func (d kv) Compact() error {
	d.store.Sync()
	return d.store.Flatten(1)
}

func (d kv) Close() error {
	return d.store.Close()
}

func New(path string, readOnly bool) (DB, error) {
	opt := badger.DefaultOptions(path)
	opt.BaseTableSize = 100 << 20
	opt.BaseLevelSize = (100 << 20) * 10
	opt.ReadOnly = readOnly

	d, err := badger.Open(opt)
	if err != nil {
		return nil, err
	}

	return &kv{
		store:      d,
		namespaces: map[string]*ns{},
	}, nil
}

func putRaw(db *badger.DB, key, val []byte) error {
	return db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
}

func genKey(ks ...[]byte) []byte {
	var ret []byte
	for _, k := range ks {
		ret = append(ret, k...)
	}
	return ret
}

func (k *kv) CreateNamespace(n []byte) (Namespace, error) {
	k.Lock()
	defer k.Unlock()

	name := string(n)

	if n, ok := k.namespaces[name]; ok {
		return n, nil
	}

	meta := kvMeta{}
	for n := range k.namespaces {
		meta.Namespaces = append(meta.Namespaces, n)
	}
	meta.Namespaces = append(meta.Namespaces, name)

	// update meta
	content, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	err = putRaw(k.store, []byte(dbMetaPrefix), content)
	if err != nil {
		return nil, err
	}

	ret := newNS(k.store, genKey([]byte(dbDataPrefix), n))
	k.namespaces[name] = ret
	return ret, nil
}

func (k *kv) DeleteNamespace(name []byte) error {
	panic("implement it")
}

type ns struct {
	sync.Mutex
	store        *badger.DB
	prefix       []byte
	doc, obj     *bucket
	otherBuckets map[string]*bucket
}

func newNS(store *badger.DB, prefix []byte) *ns {
	ret := &ns{
		store:        store,
		prefix:       prefix,
		doc:          newBucket(store, genKey(prefix, []byte(builtinBucketPrefix), []byte(builtinDocBucketName))),
		obj:          newBucket(store, genKey(prefix, []byte(builtinBucketPrefix), []byte(builtinObjectBucketName))),
		otherBuckets: map[string]*bucket{},
	}
	return ret
}

func (n *ns) CreateBucket(name []byte) (Bucket, error) {
	n.Lock()
	defer n.Unlock()

	if b, ok := n.otherBuckets[string(name)]; ok {
		return b, nil
	}

	b := newBucket(n.store, genKey(n.prefix, []byte(otherBucketPrefix), name))
	n.otherBuckets[string(name)] = b
	return b, nil
}

func (n *ns) DeleteBucket(name []byte) error {
	panic("implement it")
}

func (n *ns) DocBucket() DocBucket {
	return n.doc
}

func (n *ns) ObjectBucket() Bucket {
	return n.obj
}

type bucket struct {
	store  *badger.DB
	prefix []byte
}

func newBucket(store *badger.DB, prefix []byte) *bucket {
	return &bucket{
		store:  store,
		prefix: prefix,
	}
}

func (b *bucket) PutDoc(key []byte, item Item) error {
	content, err := bson.Marshal(item)
	if err != nil {
		return err
	}

	return b.Put(key, content)
}

func (b *bucket) GetDoc(key []byte) (Item, error) {
	val, err := b.Get(key)
	if err != nil {
		return nil, err
	}
	var item = new(Item)
	err = bson.Unmarshal(val, item)
	return *item, err
}

func (b *bucket) Find(query Query) (DocIterator, error) {
	return b.find(query)
}

func (b bucket) key(key []byte) []byte {
	return append(b.prefix, key...)
}

func (b *bucket) Put(key, val []byte) error {
	return b.store.Update(func(txn *badger.Txn) error {
		return txn.Set(b.key(key), val)
	})
}

func (b *bucket) Delete(key []byte) (err error) {
	panic("implement it")
}

func (b *bucket) Range(begin, end []byte) (Iterator, error) {
	panic("implement it")
}

func (b *bucket) Get(key []byte) (val []byte, err error) {
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

func (b *bucket) Exists(key []byte) (yes bool, err error) {
	err = b.store.View(func(txn *badger.Txn) error {
		_, err := txn.Get(b.key(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				yes = false
				return nil
			}
			return err
		}
		yes = true
		return nil
	})
	return
}

func (b *bucket) find(query Query) (*iterator, error) {
	txn := b.store.NewTransaction(false)
	opt := badger.DefaultIteratorOptions
	opt.Reverse = true
	iter := txn.NewIterator(opt)
	return &iterator{
		query:  query,
		txn:    txn,
		iter:   iter,
		prefix: b.prefix,
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
		i.iter.Seek(append(i.prefix, 'F'))
		i.init = true
	} else {
		i.iter.Next()
	}
	return i.iter.ValidForPrefix(i.prefix)
}

func (i *iterator) Key() ([]byte, error) {
	return i.iter.Item().Key(), nil
}

func (i *iterator) Value() ([]byte, error) {
	return i.iter.Item().ValueCopy(nil)
}

func (i *iterator) ValueDoc() (Item, error) {
	var item = new(Item)
	err := i.iter.Item().Value(func(val []byte) error {
		return bson.Unmarshal(val, item)
	})
	if err != nil {
		return nil, err
	}

	return *item, nil
}

func (i *iterator) Err() error {
	return nil
}

func (i *iterator) Release() error {
	i.iter.Close()
	i.txn.Discard()
	return nil
}
