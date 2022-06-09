package pkg

import "github.com/dgraph-io/badger/v3"

type Logger = badger.Logger

type dbOption struct {
	logger   Logger
	readOnly bool
}

type Option func(*dbOption)

func WithLogger(l Logger) Option {
	return func(option *dbOption) {
		option.logger = l
	}
}

func ReadOnly() Option {
	return func(option *dbOption) {
		option.readOnly = true
	}
}

func applyOptions(f []Option) *dbOption {
	opt := &dbOption{
		readOnly: false,
	}
	for _, fn := range f {
		fn(opt)
	}
	return opt
}
