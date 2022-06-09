package pkg

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dgraph-io/badger/v3"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	dbDataPrefix = "d"
	dbMetaPrefix = "m"

	builtinBucketPrefix = "b"
	otherBucketPrefix   = "o"

	builtinDocBucketName    = "d"
	builtinObjectBucketName = "o"

	inBucketKeyPrefix  = "b"
	inBucketMetaPrefix = "o"
	inBucketMetaIncKey = "id"
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

func New(path string, opts ...Option) (DB, error) {
	inOpt := applyOptions(opts)
	opt := badger.DefaultOptions(path)
	opt.BaseTableSize = 100 << 20
	opt.BaseLevelSize = (100 << 20) * 10
	opt.ReadOnly = inOpt.readOnly
	if inOpt.logger != nil {
		opt.Logger = inOpt.logger
	}

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
	ns, ok := k.namespaces[string(name)]
	if !ok {
		return fmt.Errorf("namespace %q not found", string(name))
	}
	return ns.deleteAll()
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
	b, ok := n.otherBuckets[string(name)]
	if !ok {
		return fmt.Errorf("bucket %q not exists", string(name))
	}
	return b.deleteAll()
}

func (n *ns) deleteAll() error {
	bs := []*bucket{n.doc, n.obj}
	for _, b := range n.otherBuckets {
		bs = append(bs, b)
	}
	for _, b := range bs {
		if err := b.deleteAll(); err != nil {
			return err
		}
	}
	return nil
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
	return b.find(query, true)
}

func (b bucket) key(key []byte) []byte {
	return genKey(b.prefix, []byte(inBucketKeyPrefix), key)
}

func (b *bucket) Put(key, val []byte) error {
	return b.store.Update(func(txn *badger.Txn) error {
		return txn.Set(b.key(key), val)
	})
}

func (b *bucket) PutVal(val []byte) ([]byte, error) {
	seq, err := b.store.GetSequence(genKey(b.prefix, []byte(inBucketMetaPrefix), []byte(inBucketMetaIncKey)), 1)
	if err != nil {
		return nil, err
	}
	defer seq.Release()
	n, err := seq.Next()
	if err != nil {
		return nil, err
	}

	var key [8]byte
	binary.BigEndian.PutUint64(key[:], n)
	err = b.Put(key[:], val)
	if err != nil {
		return nil, err
	}
	return key[:], nil
}

func (b *bucket) Delete(key []byte) (err error) {
	return b.store.Update(func(txn *badger.Txn) error {
		return txn.Delete(b.key(key))
	})
}

func (b *bucket) Range(begin, end []byte, reverse bool) (Iterator, error) {
	if begin != nil || end != nil {
		panic("not implemented")
	}
	return b.find(Query{}, reverse)
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

func (b *bucket) deleteAll() error {
	return b.keys(func(key []byte) bool {
		_ = b.store.Update(func(txn *badger.Txn) error {
			return txn.Delete(key)
		})
		return true
	})
}

func (b *bucket) keys(fn func([]byte) (goon bool)) error {
	return b.store.View(func(txn *badger.Txn) error {
		opt := badger.DefaultIteratorOptions
		opt.PrefetchValues = false
		it := txn.NewIterator(opt)
		defer it.Close()
		for it.Seek(b.prefix); it.ValidForPrefix(b.prefix); it.Next() {
			if !fn(it.Item().Key()) {
				break
			}
		}
		return nil
	})
}

func (b *bucket) find(query Query, reverse bool) (*iterator, error) {
	txn := b.store.NewTransaction(false)
	opt := badger.DefaultIteratorOptions
	opt.Reverse = reverse
	iter := txn.NewIterator(opt)
	return &iterator{
		query:   query,
		txn:     txn,
		iter:    iter,
		reverse: reverse,
		prefix:  genKey(b.prefix, []byte(inBucketKeyPrefix)),
	}, nil
}

type iterator struct {
	query   Query
	init    bool
	prefix  []byte
	reverse bool

	iter *badger.Iterator
	txn  *badger.Txn
}

func (i *iterator) Next() bool {
	if !i.init {
		if i.reverse {
			i.iter.Seek(append(i.prefix, 0xFF))
		} else {
			i.iter.Seek(i.prefix)
		}
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
