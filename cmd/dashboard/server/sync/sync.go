package sync

import (
	"context"
	"fmt"

	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"

	"github.com/sincaw/archivedb/cmd/dashboard/server/common"
	"github.com/sincaw/archivedb/cmd/dashboard/server/utils"
	"github.com/sincaw/archivedb/pkg"
)

type Sync struct {
	ctx context.Context

	ns        pkg.Namespace
	favBucket pkg.Bucket

	config common.SyncerConfig

	httpCli  *httpCli
	cron     *cron.Cron
	notifyCh chan struct{}
}

// New Sync instance with db ns and its configuration
func New(ctx context.Context, ns pkg.Namespace, config common.SyncerConfig) (*Sync, error) {
	if err := config.ContentTypes.ImageQuality.Valid(); err != nil {
		return nil, err
	}
	if err := config.ContentTypes.VideoQuality.Valid(); err != nil {
		return nil, err
	}

	// defaults
	if config.StartPage < 1 {
		config.StartPage = 1
	}

	cli, err := NewWithHeader(map[string]string{"cookie": config.Cookie})
	if err != nil {
		return nil, err
	}

	notifyCh := make(chan struct{}, 1)
	c := cron.New()
	_, err = c.AddFunc(config.Cron, func() {
		select {
		case notifyCh <- struct{}{}:
		default:
		}
	})
	if err != nil {
		return nil, err
	}

	fav, err := ns.CreateBucket([]byte(common.WeiboFavIndexBucket))
	if err != nil {
		return nil, err
	}

	return &Sync{
		ctx: ctx,

		ns:        ns,
		favBucket: fav,

		config: config,

		httpCli:  cli,
		cron:     c,
		notifyCh: notifyCh,
	}, nil
}

func (s *Sync) Start() {
	// sync right now
	s.notifyCh <- struct{}{}

	s.cron.Start()
	defer s.cron.Stop()

	for {
		select {
		case <-s.notifyCh:
			s.syncOnce()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Sync) syncOnce() {
	page := s.config.StartPage

	defer func() {
		fmt.Println("done")
	}()

	for {
		fmt.Printf("page %d\n", page)
		url := fmt.Sprintf("https://weibo.com/ajax/favorites/all_fav?uid=%s&page=%d", s.config.Uid, page)
		page++
		resp, err := s.httpCli.Get(url)
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
			stop, err := s.saveTweet(it)
			if err != nil {
				zap.S().Error(err)
				return
			}
			if stop {
				return
			}
		}
	}
}

// saveTweet save doc(it) and its media resources (images, video)
func (s *Sync) saveTweet(it pkg.Item) (stop bool, err error) {
	var (
		doc = s.ns.DocBucket()
		oss = s.ns.ObjectBucket()
		fav = s.favBucket
	)

	key := []byte(utils.DocId(it))
	yes, err := doc.Exists(key)
	if err != nil {
		return
	}
	if yes {
		if s.config.IncrementalMode {
			return true, nil
		}
		zap.S().Infof("skip with key %q", string(key))
		return
	}
	urls, err := s.filterImageResources(it)
	if err != nil {
		panic(err)
	}
	images, err := s.httpCli.GetImages(urls)
	if err != nil {
		return
	}
	for n, img := range images {
		err = oss.Put([]byte(n), img)
		if err != nil {
			return
		}
	}

	// check if video exists to prevent huge network usage
	yes, err = oss.Exists(key)
	if err != nil {
		return
	}

	if !yes {
		video, err := s.httpCli.FetchVideoIfNeeded(it)
		if err != nil {
			fmt.Printf("fetch video %v\n", err)
		}
		if len(video) != 0 {
			// a tweet has only one video, save video use tweet key
			err = oss.Put(key, video)
			if err != nil {
				return false, err
			}
		}
	}

	err = s.httpCli.FetchLongTextIfNeeded(it)
	if err != nil {
		zap.S().Error(err)
	}
	t := utils.OriginTweet(it)
	t[common.ExtraImagesKey] = urls
	err = doc.PutDoc(key, it)
	if err != nil {
		zap.S().Error(err)
		return false, err
	}

	// save fav
	_, err = fav.PutVal(key)

	return
}

type resource struct {
	Thumb  string `json:"thumb"`
	Origin string `json:"origin"`
}

// filterImageResources get image urls using config rule
func (s Sync) filterImageResources(item pkg.Item) (map[string]resource, error) {
	item = utils.OriginTweet(item)
	it, ok := item[utils.PicInfoKey]
	if !ok {
		return nil, nil
	}
	var (
		picInfo   = it.(bson.M)
		imgQ      = s.config.ContentTypes.ImageQuality
		withThumb = s.config.ContentTypes.Thumbnail
	)

	ret := map[string]resource{}
	for key, i := range picInfo {
		u, tu, err := imgQ.Get(i.(pkg.Item), withThumb)
		if err != nil {
			return nil, err
		}
		ret[key] = resource{
			Thumb:  tu,
			Origin: u,
		}
	}
	return ret, nil
}
