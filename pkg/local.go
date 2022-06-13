package pkg

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dgraph-io/badger/v3"
	"go.mongodb.org/mongo-driver/bson"
)

type kv struct {
	sync.Mutex
	store      *badger.DB
	namespaces map[string]*ns
}

type kvMeta struct {
	Namespaces []string `json:"namespaces"`
}

func (d *kv) Compact() error {
	d.store.Sync()
	return d.store.Flatten(1)
}

func (d *kv) Close() error {
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

func getRaw(db *badger.DB, key []byte) (val []byte, err error) {
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})
	return
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
	err = putRaw(k.store, []byte{dbMetaPrefix}, content)
	if err != nil {
		return nil, err
	}

	ret := newNS(k.store, mergeBytes([]byte{dbDataPrefix}, n))
	err = ret.loadMetas()
	if err != nil {
		return nil, err
	}

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
	store           *badger.DB
	prefix          []byte
	doc, obj, chunk *bucket
	otherBuckets    map[string]*bucket
}

func newNS(store *badger.DB, prefix []byte) *ns {
	chunk := newBucket(store, mergeBytes(prefix, []byte{nsBuiltinBucketPrefix}, []byte(builtinChunkBucketName)), nil)
	ret := &ns{
		store:        store,
		prefix:       prefix,
		doc:          newBucket(store, mergeBytes(prefix, []byte{nsBuiltinBucketPrefix}, []byte(builtinDocBucketName)), chunk),
		obj:          newBucket(store, mergeBytes(prefix, []byte{nsBuiltinBucketPrefix}, []byte(builtinObjectBucketName)), chunk),
		otherBuckets: map[string]*bucket{},
	}
	return ret
}

func (n *ns) CreateBucket(name []byte) (Bucket, error) {
	n.Lock()
	if b, ok := n.otherBuckets[string(name)]; ok {
		n.Unlock()
		return b, nil
	}
	b := newBucket(n.store, mergeBytes(n.prefix, []byte{nsOtherBucketPrefix}, name), n.chunk)
	n.otherBuckets[string(name)] = b
	n.Unlock()

	err := n.saveMetas()
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (n *ns) DeleteBucket(name []byte) error {
	b, ok := n.otherBuckets[string(name)]
	if !ok {
		return fmt.Errorf("bucket %q not exists", string(name))
	}
	return b.deleteAll()
}

func (n *ns) ListBucket() ([]string, error) {
	buckets := make([]string, 0, len(n.otherBuckets))
	n.Lock()
	for n := range n.otherBuckets {
		buckets = append(buckets, n)
	}
	n.Unlock()
	return buckets, nil
}

func (n *ns) saveMetas() error {
	buckets, err := n.ListBucket()
	if err != nil {
		return err
	}

	content, err := json.Marshal(buckets)
	if err != nil {
		return err
	}
	return putRaw(n.store, mergeBytes(n.prefix, []byte{nsMetaPrefix}, []byte(nsBucketListKey)), content)
}

func (n *ns) loadMetas() error {
	content, err := getRaw(n.store, mergeBytes(n.prefix, []byte{nsMetaPrefix}, []byte(nsBucketListKey)))
	if err != nil {
		if err == ErrKeyNotFound {
			return nil
		}
		return err
	}
	buckets := make([]string, 0)
	err = json.Unmarshal(content, &buckets)
	n.Lock()
	for _, b := range buckets {
		if _, ok := n.otherBuckets[b]; !ok {
			bu := newBucket(n.store, mergeBytes(n.prefix, []byte{nsOtherBucketPrefix}, []byte(b)), n.chunk)
			n.otherBuckets[b] = bu
		}
	}
	n.Unlock()
	return nil
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
	chunk  *bucket
	prefix []byte
}

func newBucket(store *badger.DB, prefix []byte, chunk *bucket) *bucket {
	return &bucket{
		store:  store,
		prefix: prefix,
		chunk:  chunk,
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
	val, _, err := b.Get(key)
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
	return mergeBytes(b.prefix, []byte{bucketKeyPrefix}, key)
}

// Put saves val by key
// value format:
// | 1byte (has meta) | 4byte (uint32: meta len) | meta | value |
// value will saved in chunk bucket when meta.ChunkSize > 0
// e.g.
// Value without meta: [0, value...]
// Value with meta and 1 piece [1, meta len, meta..., value...]
// Value with meta and n piece [1, meta len, meta...] and ([chunk 0] ... [chunk n]) in chunk bucket
func (b *bucket) Put(key, val []byte, opts ...PutOption) error {
	opt := applyPutOptions(opts)
	if opt.meta == nil {
		return b.put(key, mergeBytes([]byte{valueWithoutMeta}, val))
	}

	if opt.meta.ChunkSize < 0 {
		return fmt.Errorf("invalid meta")
	}

	opt.meta.TotalLen = len(val)
	if opt.meta.ChunkSize == 0 || opt.meta.ChunkSize > opt.meta.TotalLen {
		return b.putWithMeta(key, val, opt.meta)
	}

	var chunks [][]byte
	for i := 0; i < len(val); i += opt.meta.ChunkSize {
		end := i + opt.meta.ChunkSize
		if end > len(val) {
			end = len(val)
		}
		chunks = append(chunks, val[i:end])
	}

	for _, chunk := range chunks {
		k, err := b.chunk.PutVal(chunk)
		if err != nil {
			return err
		}
		opt.meta.Chunks = append(opt.meta.Chunks, k)
	}
	return b.putWithMeta(key, nil, opt.meta)
}

func (b *bucket) put(key, val []byte) error {
	return b.store.Update(func(txn *badger.Txn) error {
		return txn.Set(b.key(key), val)
	})
}

func (b *bucket) putWithMeta(key, val []byte, meta *Meta) error {
	m, err := bson.Marshal(meta)
	if err != nil {
		return err
	}

	metaLen := make([]byte, 4)
	binary.BigEndian.PutUint32(metaLen, uint32(len(m)))
	content := mergeBytes([]byte{valueWithMeta}, metaLen, m, val)
	return b.put(key, content)
}

func (b *bucket) PutVal(val []byte, opts ...PutOption) ([]byte, error) {
	seq, err := b.store.GetSequence(mergeBytes(b.prefix, []byte{bucketMetaPrefix}, []byte(inBucketMetaIncKey)), 1)
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
		item, err := txn.Get(b.key(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}
			return err
		}
		val, err := item.ValueCopy(nil)
		_, meta, err := unpackValue(val)
		if meta == nil || len(meta.Chunks) == 0 {
			return txn.Delete(b.key(key))
		}

		for _, c := range meta.Chunks {
			err = b.chunk.Delete(c)
			if err != nil {
				return err
			}
		}
		return txn.Delete(b.key(key))
	})
}

func (b *bucket) Range(begin, end []byte, reverse bool) (Iterator, error) {
	if begin != nil || end != nil {
		panic("not implemented")
	}
	return b.find(Query{}, reverse)
}

func unpackValue(val []byte) ([]byte, *Meta, error) {
	if val == nil {
		return nil, nil, nil
	}
	m := val[0]
	if m == valueWithoutMeta {
		return val[1:], nil, nil
	}

	metaLen := binary.BigEndian.Uint32(val[1:])
	start := 1 + 4
	metaBuf := val[start : start+int(metaLen)]
	var meta = new(Meta)
	err := bson.Unmarshal(metaBuf, meta)
	if err != nil {
		return nil, nil, fmt.Errorf("unmarshal meta fail with err: %v", err)
	}

	return val[1+4+metaLen:], meta, nil
}

func (b *bucket) Get(key []byte) (val []byte, meta *Meta, err error) {
	err = b.store.View(func(txn *badger.Txn) error {
		item, err := txn.Get(b.key(key))
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}
		val, meta, err = unpackValue(val)
		if err != nil {
			return err
		}
		// no chunks
		if meta == nil || len(meta.Chunks) == 0 {
			return nil
		}

		// merge all chunks
		for _, k := range meta.Chunks {
			err = b.chunk.get(k, func(v []byte) {
				val = append(val, v...)
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return
}

func (b *bucket) get(key []byte, fn func(val []byte)) error {
	return b.store.View(func(txn *badger.Txn) error {
		item, err := txn.Get(b.key(key))
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			v, _, err := unpackValue(val)
			if err != nil {
				return err
			}
			fn(v)
			return nil
		})
		return err
	})
}

func (b *bucket) GetAt(key, buf []byte, offset int) (n int, err error) {
	meta, err := b.GetMeta(key)
	if err != nil {
		return 0, err
	}
	if meta == nil || len(meta.Chunks) == 0 {
		v, _, err := b.Get(key)
		if err != nil {
			return 0, err
		}
		if offset >= len(v) {
			return 0, fmt.Errorf("out of range [0,%d)", meta.TotalLen)
		}
		n = copy(buf, v[offset:])
		return n, nil
	}

	if offset > meta.TotalLen || offset < 0 {
		return 0, fmt.Errorf("out of range [0,%d) with meta", meta.TotalLen)
	}

	var (
		startIdx = offset / meta.ChunkSize
		endIdx   = len(buf)/meta.ChunkSize + startIdx + 1
	)
	if endIdx >= len(meta.Chunks) {
		endIdx = len(meta.Chunks)
	}

	for i := startIdx; i < endIdx; i++ {
		valOff := 0
		if i == startIdx {
			valOff = offset - i*meta.ChunkSize
		}
		err = b.chunk.get(meta.Chunks[i], func(val []byte) {
			n += copy(buf[n:], val[valOff:])
		})
		if err != nil {
			return 0, err
		}
		if n >= len(buf) {
			break
		}
	}
	if n > len(buf) {
		n = len(buf)
	}

	return n, nil
}

func (b *bucket) GetMeta(key []byte) (meta *Meta, err error) {
	err = b.store.View(func(txn *badger.Txn) error {
		item, err := txn.Get(b.key(key))
		if err != nil {
			return err
		}
		val, err := item.ValueCopy(nil)
		_, meta, err = unpackValue(val)
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
		prefix:  mergeBytes(b.prefix, []byte{bucketKeyPrefix}),
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
	v, err := i.iter.Item().ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	v, _, err = unpackValue(v)
	return v, err
}

func (i *iterator) ValueDoc() (Item, error) {
	var item = new(Item)
	err := i.iter.Item().Value(func(val []byte) error {
		val, _, err := unpackValue(val)
		if err != nil {
			return err
		}
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
