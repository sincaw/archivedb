package local

import (
	"os"
	"testing"

	"github.com/dgraph-io/badger/v3"
	"github.com/stretchr/testify/require"

	"github.com/jialeicui/archivedb/pkg"
)

const (
	tempDirPattern = "archive"
)

func TestLocal(t *testing.T) {
	path, err := os.MkdirTemp("", tempDirPattern)
	require.Nil(t, err)
	defer os.RemoveAll(path)

	db, err := New(path)
	require.Nil(t, err)
	defer db.Close()

	item := &pkg.Item{"foo": "bar", "baz": pkg.Item{"bar": "foo"}}
	err = db.Put(item)
	require.Nil(t, err)

	iter, err := db.Find(pkg.Query{})
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
