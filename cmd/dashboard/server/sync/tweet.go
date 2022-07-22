package sync

import (
	"encoding/json"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/sincaw/archivedb/cmd/dashboard/server/common"
	"github.com/sincaw/archivedb/cmd/dashboard/server/utils"
	"github.com/sincaw/archivedb/pkg"
)

// saveTweet save doc(it) and its media resources (images, video)
func (s *Sync) saveTweet(ns pkg.Namespace, it pkg.Item, conf common.ContentTypes, incMode bool) (stop bool, err error) {
	var (
		doc = ns.DocBucket()
		oss = ns.ObjectBucket()
		key = []byte(utils.DocId(it))
		l   = logger.With("weibo id", string(key))
	)

	l.Debugf("process weibo id %q", string(key))

	yes, err := doc.Exists(key)
	if err != nil {
		return
	}
	if yes {
		if incMode {
			return true, nil
		}
		l.Infof("skip with key %q", string(key))
		return
	}

	err = s.saveImagesForTweet(it, oss, conf.ImageQuality, conf.Thumbnail, l)
	if err != nil {
		return
	}

	err = s.saveVideoForTweet(it, oss, conf.VideoQuality, l)
	if err != nil {
		return
	}

	err = FetchLongTextIfNeeded(s.httpCli, it)
	if err != nil {
		l.Errorf("fetch long text fail %v", err)
	}

	err = doc.PutDoc(key, it)
	if err != nil {
		l.Errorf("save tweet doc fail %v", err)
		return
	}

	return
}

func (s *Sync) saveImagesForTweet(tweet pkg.Item, oss pkg.Bucket, q common.ImageQuality, withThumb bool, l *zap.SugaredLogger) (err error) {
	urls, err := s.getImageUrls(tweet, q, withThumb)
	if err != nil {
		l.Errorf("get image url list fail %v", err)
		return
	}

	l.Debug("try getting images")

	images, err := GetImages(s.httpCli, urls)
	if err != nil {
		l.Errorf("get images fail %v", err)
		return
	}

	for n, img := range images {
		err = oss.Put([]byte(n), img, pkg.WithMeta(&pkg.Meta{Mime: common.MimeImage}))
		if err != nil {
			return err
		}
	}

	t := utils.OriginTweet(tweet)
	t[common.ExtraImagesKey] = urls

	return
}

func (s *Sync) saveVideoForTweet(tweet pkg.Item, oss pkg.Bucket, q common.VideoQuality, l *zap.SugaredLogger) (err error) {
	key := []byte(utils.DocId(tweet))

	l.Debug("try getting video")
	// check if video exists to prevent huge network usage
	yes, err := oss.Exists(key)
	if err != nil {
		return
	}

	if !yes {
		video, err := FetchVideoIfNeeded(s.httpCli, tweet, q)
		if err != nil {
			l.Errorf("fetch video fail %v", err)
		}
		if len(video) != 0 {
			// a tweet has only one video, save video use tweet key
			err = oss.Put(key, video, pkg.WithMeta(&pkg.Meta{Mime: common.MimeVideo, ChunkSize: 5 * 1024 * 1024}))
			if err != nil {
				return err
			}
			t := utils.OriginTweet(tweet)
			t[common.ExtraVideoKey] = fmt.Sprintf("%s.mp4", string(key))
		}
	}
	return nil
}

type resource struct {
	Thumb  string `json:"thumb"`
	Origin string `json:"origin"`
	Live   string `json:"live"`
}

// getImageUrls get image urls using config rule
func (s Sync) getImageUrls(item pkg.Item, q common.ImageQuality, withThumb bool) (map[string]resource, error) {
	item = utils.OriginTweet(item)
	it, ok := item[utils.PicInfoKey]
	if !ok {
		return nil, nil
	}
	var (
		picInfo = it.(bson.M)
	)

	ret := map[string]resource{}
	for key, i := range picInfo {
		u, tu, lu, err := q.Get(i.(pkg.Item), withThumb)
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
