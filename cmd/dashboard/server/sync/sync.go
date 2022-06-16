package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/sync/errgroup"

	"github.com/sincaw/archivedb/cmd/dashboard/server/common"
	"github.com/sincaw/archivedb/cmd/dashboard/server/utils"
	"github.com/sincaw/archivedb/pkg"
)

var (
	logger = utils.Logger()
)

const (
	FavAPI = "https://weibo.com/ajax/favorites/all_fav?uid=%s&page=%d"
)

type Sync struct {
	ctx context.Context

	ns        pkg.Namespace
	favBucket pkg.Bucket

	config common.SyncerConfig

	httpCli  *common.HttpCli
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

	cli, err := common.NewWithHeader(map[string]string{"cookie": config.Cookie})
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

func (s *Sync) Accept(cookie string) {
	cli, err := common.NewWithHeader(map[string]string{"cookie": cookie})
	if err != nil {
		logger.Error("update cookie fail ", err)
		return
	}
	s.httpCli = cli
	logger.Info("update cookie success, trigger sync")
	s.notifyCh <- struct{}{}
}

func (s *Sync) syncOnce() {
	page := s.config.StartPage

	defer func() {
		logger.Info("sync done")
	}()

	for {
		logger.Infof("page %d", page)
		url := fmt.Sprintf(FavAPI, s.config.Uid, page)
		resp, err := s.httpCli.Get(url)
		if err != nil {
			logger.Error(err)
			return
		}
		logger.Debugf("fetch page %d done", page)

		page++

		item := pkg.Item{}
		err = bson.UnmarshalExtJSON(resp, true, &item)
		if err != nil {
			logger.Errorf(fmt.Sprintf("unmarshal content fail, err %v", err))
			return
		}
		ok := item["ok"]
		if ok != int32(1) {
			logger.Errorf(fmt.Sprintf("invalid content: %q", string(resp)))
			return
		}

		items := item["data"].(bson.A)
		logger.Debugf("parse %d items", len(items))

		if len(items) == 0 {
			return
		}

		for _, i := range items {
			it := i.(pkg.Item)
			stop, err := s.saveTweet(it)
			if err != nil {
				logger.Error(err)
				continue
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
	l := logger.With("weibo id", string(key))
	l.Debugf("process weibo id %q", string(key))

	yes, err := doc.Exists(key)
	if err != nil {
		return
	}
	if yes {
		if s.config.IncrementalMode {
			return true, nil
		}
		l.Infof("skip with key %q", string(key))
		return
	}
	urls, err := s.filterImageResources(it)
	if err != nil {
		l.Panic(err)
	}

	l.Debug("get images")
	images, err := GetImages(s.httpCli, urls)
	if err != nil {
		l.Errorf("get images fail: %v", err)
		return
	}
	for n, img := range images {
		err = oss.Put([]byte(n), img, pkg.WithMeta(&pkg.Meta{Mime: common.MimeImage}))
		if err != nil {
			return
		}
	}

	// check if video exists to prevent huge network usage
	yes, err = oss.Exists(key)
	if err != nil {
		return
	}

	t := utils.OriginTweet(it)

	if !yes {
		video, err := FetchVideoIfNeeded(s.httpCli, it, s.config.ContentTypes.VideoQuality)
		if err != nil {
			l.Errorf("fetch video %v", err)
		}
		if len(video) != 0 {
			// a tweet has only one video, save video use tweet key
			err = oss.Put(key, video, pkg.WithMeta(&pkg.Meta{Mime: common.MimeVideo, ChunkSize: 5 * 1024 * 1024}))
			if err != nil {
				return false, err
			}
			t[common.ExtraVideoKey] = fmt.Sprintf("%s.mp4", string(key))
		}
	}

	err = FetchLongTextIfNeeded(s.httpCli, it)
	if err != nil {
		logger.Error(err)
	}
	t[common.ExtraImagesKey] = urls
	err = doc.PutDoc(key, it)
	if err != nil {
		logger.Error(err)
		return false, err
	}

	// save fav
	_, err = fav.PutVal(key)

	return
}

type resource struct {
	Thumb  string `json:"thumb"`
	Origin string `json:"origin"`
	Live   string `json:"live"`
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
		u, tu, lu, err := imgQ.Get(i.(pkg.Item), withThumb)
		if err != nil {
			return nil, err
		}
		ret[key] = resource{
			Thumb:  tu,
			Origin: u,
			Live:   lu,
		}
	}
	return ret, nil
}
func GetImages(cli *common.HttpCli, rcs map[string]resource) (pkg.Resources, error) {
	if len(rcs) == 0 {
		return nil, nil
	}
	var (
		rc   = make(pkg.Resources)
		mu   sync.Mutex
		eg   errgroup.Group
		urls = map[string]string{}
	)

	// get all unique urls
	for k, i := range rcs {
		urls[k+"-thumb"] = i.Thumb
		urls[k+"-live"] = i.Live
		urls[k] = i.Origin
	}

	for n, url := range urls {
		if url == "" {
			continue
		}
		n := n
		url := url
		eg.Go(func() error {
			resp, err := cli.Get(url)
			if err != nil {
				logger.Errorf("ignore, get image fail: %v", err)
				return nil
			}
			mu.Lock()
			rc[n] = resp
			mu.Unlock()
			return nil
		})
	}
	return rc, eg.Wait()
}

func FetchLongTextIfNeeded(cli *common.HttpCli, item pkg.Item) error {
	item = utils.OriginTweet(item)
	if _, ok := item["continue_tag"]; !ok {
		return nil
	}

	id, ok := item["mblogid"]
	if !ok {
		return fmt.Errorf("no valid mblog id for %q", utils.DocId(item))
	}
	resp, err := cli.Get(fmt.Sprintf("https://weibo.com/ajax/statuses/longtext?id=%s", id))
	if err != nil {
		logger.Error(err)
		return err
	}

	type longResp struct {
		Ok   int `json:"ok"`
		Data struct {
			LongTextContent string `json:"longTextContent"`
		} `json:"data"`
	}
	long := new(longResp)
	err = json.Unmarshal(resp, long)
	if err != nil {
		logger.Error(err)
		return fmt.Errorf("unmarshal fail with context %s, err %v", string(resp), err)
	}
	item["text_raw"] = long.Data.LongTextContent
	return nil
}

// FetchVideoIfNeeded try parse video in doc
// It returns (nil, nil) when there is no video
// or return video content and nil error
func FetchVideoIfNeeded(cli *common.HttpCli, item pkg.Item, vq common.VideoQuality) ([]byte, error) {
	if vq == common.VideoQualityNone {
		logger.Debugf("no need to fetch video")
		return nil, nil
	}

	if !utils.HasVideo(item) {
		logger.Debugf("no video")
		return nil, nil
	}

	item = utils.OriginTweet(item)
	id, ok := item["mblogid"]
	if !ok {
		return nil, fmt.Errorf("no valid mblog id for %q", utils.DocId(item))
	}

	resp, err := cli.Get(fmt.Sprintf("https://weibo.com/ajax/statuses/show?id=%s", id))
	if err != nil {
		return nil, err
	}

	url, err := vq.Get(resp)
	if err != nil {
		return nil, err
	}
	if url == "" {
		return nil, nil
	}

	logger.Debugf("fetch video, url: %s", url)
	return cli.Get(url)
}
