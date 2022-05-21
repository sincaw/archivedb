package pkg

import (
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
	db, err = New(path, false)
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
	val, err := b.Get(k)
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
	v, err := b.Get(key)
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
	v, err := b.Get(key)
	require.Equal(t, badger.ErrKeyNotFound, err)
	require.Nil(t, v)

	v, err = doc.Get(key)
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
	_, err = ns1.DocBucket().Get(key)
	require.Equal(t, badger.ErrKeyNotFound, err)
	v, err := ns2.DocBucket().Get(key)
	require.Nil(t, err)
	require.Equal(t, val, v)
}
