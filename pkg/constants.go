package pkg

type MetaPrefix = byte

const (
	dbDataPrefix MetaPrefix = iota
	dbMetaPrefix
)

const (
	nsBuiltinBucketPrefix MetaPrefix = iota
	nsOtherBucketPrefix
	nsMetaPrefix
)

const (
	nsBucketListKey = "buckets"
)

const (
	builtinDocBucketName    = "d"
	builtinObjectBucketName = "o"
	builtinChunkBucketName  = "t"
)

const (
	bucketKeyPrefix MetaPrefix = iota
	bucketMetaPrefix
)

const inBucketMetaIncKey = "id"

const (
	valueWithoutMeta = 0
	valueWithMeta    = 1
)
