package pkg

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/dgraph-io/badger/v3"
	"github.com/stretchr/testify/require"
)

const (
	tempDirPattern = "archive"
	defaultNS      = "default"
)

func mustNewDB() (db DB, cleanFn func()) {
	path, err := os.MkdirTemp("", tempDirPattern)
	if err != nil {
		panic(err)
	}
	db, err = New(path)
	if err != nil {
		panic(err)
	}

	return db, func() {
		_ = db.Close()
		_ = os.RemoveAll(path)
	}
}

func TestNamespace(t *testing.T) {
	db, clean := mustNewDB()
	defer clean()

	var (
		bucket = []byte("bucket")
		k      = []byte("foo")
		v      = []byte("bar")
	)

	ns, err := db.CreateNamespace([]byte("name"))
	require.Nil(t, err)
	b, err := ns.CreateBucket(bucket)
	require.Nil(t, err)

	ns2, err := db.CreateNamespace([]byte("another-ns"))
	require.Nil(t, err)
	b2, err := ns2.CreateBucket(bucket)
	require.Nil(t, err)

	require.Nil(t, b.Put(k, v))
	yes, err := b.Exists(k)
	require.Nil(t, err)
	require.True(t, yes)

	// same key can not found in another ns
	yes, err = b2.Exists(k)
	require.Nil(t, err)
	require.False(t, yes)
}

func mustGetDefaultNamespace(db DB) Namespace {
	ns, err := db.CreateNamespace([]byte(defaultNS))
	if err != nil {
		panic(err)
	}
	return ns
}

func TestBucket(t *testing.T) {
	db, clean := mustNewDB()
	defer clean()
	ns := mustGetDefaultNamespace(db)

	// create bucket with builtin bucket name
	b, err := ns.CreateBucket([]byte(builtinDocBucketName))
	require.Nil(t, err)

	var (
		k = []byte("foo")
		v = []byte("bar")
	)
	require.Nil(t, b.Put(k, v))
	val, _, err := b.Get(k)
	require.Nil(t, err)
	require.Equal(t, v, val)

	// check builtin bucket
	for _, b := range []Bucket{ns.DocBucket(), ns.ObjectBucket()} {
		// expect not found
		yes, err := b.Exists(k)
		require.Nil(t, err)
		require.False(t, yes)
	}
}

func TestPutAndGetDoc(t *testing.T) {
	db, clean := mustNewDB()
	defer clean()

	var (
		b    = mustGetDefaultNamespace(db).DocBucket()
		key  = []byte("key1")
		item = Item{"foo": "bar", "baz": Item{"bar": "foo"}}
	)

	err := b.PutDoc(key, item)
	require.Nil(t, err)

	v, err := b.GetDoc(key)
	require.Nil(t, err)
	require.Equal(t, item, v)
}

func TestDeleteByKey(t *testing.T) {
	db, clean := mustNewDB()
	defer clean()

	var (
		key          = []byte("foo")
		val          = []byte("bar")
		keyNotExists = []byte("baz")
		b            = mustGetDefaultNamespace(db).DocBucket()
	)
	require.Nil(t, b.Put(key, val))
	require.Nil(t, b.Delete(keyNotExists))
	require.Nil(t, b.Delete(key))
	v, _, err := b.Get(key)
	require.Equal(t, badger.ErrKeyNotFound, err)
	require.Nil(t, v)
}

func TestDeleteByBucket(t *testing.T) {
	db, clean := mustNewDB()
	defer clean()

	var (
		key    = []byte("foo")
		val    = []byte("bar")
		bucket = []byte("baz")
		ns     = mustGetDefaultNamespace(db)
		doc    = ns.DocBucket()
	)

	b, err := ns.CreateBucket(bucket)
	require.Nil(t, err)
	require.Nil(t, b.Put(key, val))
	require.Nil(t, doc.Put(key, val))

	require.Nil(t, ns.DeleteBucket(bucket))
	v, _, err := b.Get(key)
	require.Equal(t, badger.ErrKeyNotFound, err)
	require.Nil(t, v)

	v, _, err = doc.Get(key)
	require.Nil(t, err)
	require.Equal(t, val, v)
}

func TestDeleteByNamespace(t *testing.T) {
	db, clean := mustNewDB()
	defer clean()

	var (
		key = []byte("foo")
		val = []byte("bar")
	)

	ns1, err := db.CreateNamespace([]byte("ns1"))
	require.Nil(t, err)
	ns2, err := db.CreateNamespace([]byte("ns2"))
	require.Nil(t, err)

	require.Nil(t, ns1.DocBucket().Put(key, val))
	require.Nil(t, ns2.DocBucket().Put(key, val))
	require.Nil(t, db.DeleteNamespace([]byte("ns1")))
	_, _, err = ns1.DocBucket().Get(key)
	require.Equal(t, badger.ErrKeyNotFound, err)
	v, _, err := ns2.DocBucket().Get(key)
	require.Nil(t, err)
	require.Equal(t, val, v)
}

func TestPutVal(t *testing.T) {
	db, clean := mustNewDB()
	defer clean()
	var (
		ns     = mustGetDefaultNamespace(db)
		b, err = ns.CreateBucket([]byte("bucket"))
		val1   = []byte("foo")
		val2   = []byte("bar")
	)

	require.Nil(t, err)

	key1, err := b.PutVal(val1)
	require.Nil(t, err)
	key2, err := b.PutVal(val2)
	require.Nil(t, err)
	require.True(t, bytes.Compare(key2, key1) > 0)

	val, _, err := b.Get(key1)
	require.Nil(t, err)
	require.Equal(t, val1, val)

	val, _, err = b.Get(key2)
	require.Nil(t, err)
	require.Equal(t, val2, val)

	// test range
	it, err := b.Range(nil, nil, false)
	require.Nil(t, err)
	defer it.Release()
	for it.Next() {
		fmt.Println(it.Key())
	}
}

func TestMeta(t *testing.T) {
	db, clean := mustNewDB()
	defer clean()

	var (
		b    = mustGetDefaultNamespace(db).ObjectBucket()
		key  = []byte("key1")
		val  = []byte("foo")
		mime = "mp4"
	)

	err := b.Put(key, val, WithMeta(&Meta{Mime: mime}))
	require.Nil(t, err)

	v, m, err := b.Get(key)
	require.Nil(t, err)
	require.Equal(t, val, v)
	require.Equal(t, len(val), m.TotalLen)
	require.Equal(t, mime, m.Mime)

	err = b.Put(key, val, WithMeta(&Meta{Mime: mime, ChunkSize: 1}))
	require.Nil(t, err)

	v, m, err = b.Get(key)
	require.Nil(t, err)
	require.Equal(t, val, v)
	require.Equal(t, len(val), m.TotalLen)
	require.Equal(t, mime, m.Mime)
	require.True(t, len(m.Chunks) > 0)

	err = b.Delete(key)
	require.Nil(t, err)
	for _, c := range m.Chunks {
		_, _, err = b.(*bucket).chunk.Get(c)
		require.Equal(t, badger.ErrKeyNotFound, err)
	}
}

func TestGetAt(t *testing.T) {
	db, clean := mustNewDB()
	defer clean()

	var (
		b   = mustGetDefaultNamespace(db).ObjectBucket()
		key = []byte("key1")
		val = []byte("foo")
	)

	err := b.Put(key, val)
	require.Nil(t, err)

	buf := make([]byte, 2)
	// from the beginning
	n, err := b.GetAt(key, buf, 0)
	require.Nil(t, err)
	require.Equal(t, len(buf), n)
	require.Equal(t, val[0:0+len(buf)], buf)

	// from the middle
	n, err = b.GetAt(key, buf, 1)
	require.Nil(t, err)
	require.Equal(t, len(buf), n)
	require.Equal(t, val[1:1+len(buf)], buf)

	// to the end
	n, err = b.GetAt(key, buf, 2)
	require.Nil(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, val[2:], buf[:n])
}
