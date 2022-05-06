package sync

import (
	"fmt"

	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"

	"github.com/sincaw/archivedb/cmd/dashboard/server/utils"
	"github.com/sincaw/archivedb/pkg"
)


type Config struct {
	Uid    string `yaml:"uid"`
	Cookie string `yaml:"cookie"`
	Cron   string `yaml:"cron"`

	StartPage       int  `yaml:"startPage"`
	IncrementalMode bool `yaml:"incrementalMode"`
}

type Sync struct {
	ns     pkg.Namespace
	config Config

	cron     *cron.Cron
	notifyCh chan struct{}
}

func New(ns pkg.Namespace, config Config) (*Sync, error) {
	// defaults
	if config.StartPage < 1 {
		config.StartPage = 1
	}

	notifyCh := make(chan struct{}, 1)
	c := cron.New()
	_, err := c.AddFunc(config.Cron, func() {
		select {
		case notifyCh <- struct{}{}:
		default:
		}
	})
	if err != nil {
		return nil, err
	}

	return &Sync{
		ns:     ns,
		config: config,

		cron:     c,
		notifyCh: notifyCh,
	}, nil
}

func (s *Sync) Start() {
	// sync right now
	s.notifyCh <- struct{}{}

	s.cron.Start()
	cli, err := NewWithHeader(map[string]string{"cookie": s.config.Cookie})
	if err != nil {
		panic(err)
	}

	for range s.notifyCh {
		s.syncOnce(cli)
	}
}

func (s *Sync) syncOnce(cli *httpCli) {
	var (
		page = s.config.StartPage
		doc  = s.ns.DocBucket()
		oss  = s.ns.ObjectBucket()
	)

	defer func() {
		fmt.Println("done")
	}()

	for {
		fmt.Printf("page %d\n", page)
		url := fmt.Sprintf("https://weibo.com/ajax/favorites/all_fav?uid=%s&page=%d", s.config.Uid, page)
		page++
		resp, err := cli.Get(url)
		if err != nil {
			panic(err)
		}

		item := pkg.Item{}
		err = bson.UnmarshalExtJSON(resp, true, &item)
		if err != nil {
			panic(fmt.Sprintf("content %s, err %v\n", string(resp), err))
		}
		ok := item["ok"]
		if ok != int32(1) {
			panic(fmt.Sprintf("invalid content: %q", string(resp)))
		}

		items := item["data"].(bson.A)
		if len(items) == 0 {
			return
		}
		for _, i := range items {
			it := i.(pkg.Item)
			key := []byte(utils.DocId(it))
			yes, err := doc.Exists(key)
			if err != nil {
				zap.S().Error(err)
				return
			}
			if yes {
				if s.config.IncrementalMode {
					return
				}
				zap.S().Infof("skip with key %q", string(key))
				continue
			}
			urls, err := utils.FilterResources(it)
			if err != nil {
				panic(err)
			}
			images, err := cli.GetImages(urls)
			if err != nil {
				panic(err)
			}
			for n, img := range images {
				err = oss.Put([]byte(n), img)
				if err != nil {
					panic(err)
				}
			}

			// check if video exists to prevent huge network usage
			yes, err = oss.Exists(key)
			if err != nil {
				return
			}
			if !yes {
				video, err := cli.FetchVideoIfNeeded(it)
				if err != nil {
					fmt.Printf("fetch video %v\n", err)
				}
				if len(video) != 0 {
					// a tweet has only one video, save video use tweet key
					err = oss.Put(key, video)
					if err != nil {
						panic(err)
					}
				}
			}

			err = cli.FetchLongTextIfNeeded(it)
			if err != nil {
				zap.S().Error(err)
			}
			err = doc.PutDoc(key, it)
			if err != nil {
				panic(err)
			}
		}
	}
}
