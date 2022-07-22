package sync

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/sincaw/archivedb/pkg"
)

func (s *Sync) syncFavorite() {
	favConf := s.config.Favorite
	page := favConf.StartPage
	if page < 1 {
		page = 1
	}

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
			stop, err := s.saveTweet(s.ns, it, favConf.ContentTypes, favConf.IncrementalMode)
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
