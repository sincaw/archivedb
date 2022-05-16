package main

import (
	"context"
	"flag"
	"io/ioutil"
	"path"
	"path/filepath"
	"sync/atomic"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/sincaw/archivedb/cmd/dashboard/server/api"
	"github.com/sincaw/archivedb/cmd/dashboard/server/common"
	"github.com/sincaw/archivedb/cmd/dashboard/server/sync"
	"github.com/sincaw/archivedb/cmd/dashboard/server/utils"
	"github.com/sincaw/archivedb/pkg"
)

const (
	configFile = ".config.yaml"
	Namespace  = "weibo"
)

func main() {
	flag.Parse()
	logger, _ := zap.NewDevelopment()

	dir, err := utils.SelfDir()
	if err != nil {
		logger.Sugar().Fatalf("get binary dir fail %v", err)
	}
	content, err := ioutil.ReadFile(path.Join(dir, configFile))
	if err != nil {
		logger.Sugar().Fatalf("read config file fail %v", err)
	}
	config := new(common.Config)
	err = yaml.Unmarshal(content, config)
	if err != nil {
		logger.Sugar().Fatalf("parse config file fail %v", err)
	}

	dbPath := config.DatabasePath
	if !filepath.IsAbs(dbPath) {
		dbPath = path.Join(dir, dbPath)
	}
	db, err := pkg.New(dbPath, false)
	if err != nil {
		logger.Sugar().Fatalf("open db fail, path %q, err %v", dbPath, err)
	}
	defer db.Close()
	ns, err := db.CreateNamespace([]byte(Namespace))
	if err != nil {
		return
	}

	var (
		reloadCh = make(chan struct{}, 1)
		reload   = int32(0)
	)
	config.OnChange(func(common.Config) error {
		select {
		case reloadCh <- struct{}{}:
		default:
		}
		return nil
	})

	for {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			for range reloadCh {
				atomic.StoreInt32(&reload, 1)
				cancel()
				return
			}
		}()

		syncer, err := sync.New(ctx, ns, config.Syncer)
		if err != nil {
			logger.Sugar().Fatalf("config syncer fail err %v", err)
		}
		go syncer.Start()

		err = api.New(ctx, ns, config).Serve()
		if !atomic.CompareAndSwapInt32(&reload, 1, 0) {
			zap.S().Error(err)
		}
	}
}
