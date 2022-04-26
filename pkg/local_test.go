package pkg

import (
	"os"
	"testing"

	"github.com/dgraph-io/badger/v3"
	"github.com/stretchr/testify/require"
)

const (
	tempDirPattern = "archive"
)

func newDB() (DB, func(), error) {
	path, err := os.MkdirTemp("", tempDirPattern)
	if err != nil {
		return nil, nil, err
	}
	db, err := New(path)

	return db, func() {
		defer os.RemoveAll(path)
		defer db.Close()
	}, err
}

func TestLocal(t *testing.T) {
	db, clean, err := newDB()
	require.Nil(t, err)
	defer clean()

	item := &Item{"foo": "bar", "baz": Item{"bar": "foo"}}
	err = db.Put(item)
	require.Nil(t, err)

	iter, err := db.Find(Query{})
	require.Nil(t, err)
	defer iter.Release()

	// insert a key without bucket prefix
	ins := db.(*kv)
	err = ins.store.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("foo"), []byte("bar"))
	})
	require.Nil(t, err)

	count := 0
	for iter.Next() {
		val, err := iter.Value()
		require.Nil(t, err)
		require.Equal(t, item, val)
		count++
	}
	require.Equal(t, 1, count)
}

func TestPutAndGet(t *testing.T) {
	db, clean, err := newDB()
	require.Nil(t, err)
	defer clean()

	var (
		key = []byte("foo")
		val = &Item{"abc": "def"}
	)
	require.Nil(t, db.Put(val, WithKey(key)))
	item, err := db.Get(key)
	require.Nil(t, err)
	require.Equal(t, val, item)
}

func TestResources(t *testing.T) {
	db, clean, err := newDB()
	require.Nil(t, err)
	defer clean()

	var (
		key       = []byte("foo")
		val       = &Item{"abc": "def"}
		resources = Resources{"bar": []byte("baz")}
	)
	require.Nil(t, db.Put(val, WithKey(key), WithResources(resources)))
	// validate value doc
	item, err := db.Get(key)
	require.Nil(t, err)
	require.Equal(t, val, item)
	// validate resources
	rc, err := db.GetResources(key)
	require.Nil(t, err)
	require.Equal(t, resources, rc)
}
