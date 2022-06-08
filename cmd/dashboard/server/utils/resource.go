package utils

import (
	"fmt"
	"net/url"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"

	"github.com/sincaw/archivedb/pkg"
)

const (
	retweet     = "retweeted_status"
	PicInfoKey  = "pic_infos"
	originalKey = "original"
)

const (
	VideoResourceKey = "video"
)

var (
	errNoPic = fmt.Errorf("no pic info")
)

func DocId(item pkg.Item) string {
	return item["idstr"].(string)
}

func TextRaw(item pkg.Item) string {
	return item["text_raw"].(string)
}

func OriginTweet(item pkg.Item) pkg.Item {
	if _, ok := item[retweet]; ok {
		item = item[retweet].(pkg.Item)
	}
	return item
}

func HasVideo(item pkg.Item) bool {
	if _, ok := item["page_info"]; ok {
		if _, ok := item["page_info"].(pkg.Item)["media_info"]; ok {
			return true
		}
	}
	return false
}

func mapResource(item pkg.Item, fn func(it pkg.Item) error) error {
	item = OriginTweet(item)
	it, ok := item[PicInfoKey]
	if !ok {
		return errNoPic
	}
	picInfo := it.(bson.M)

	for _, i := range picInfo {
		pic := i.(pkg.Item)[originalKey]
		err := fn(pic.(pkg.Item))
		if err != nil {
			return err
		}
	}
	return nil
}
func FilterResources(item pkg.Item) ([]string, error) {
	var rc []string
	err := mapResource(item, func(it pkg.Item) error {
		url := it["url"].(string)
		rc = append(rc, url)
		return nil
	})
	// return empty array when no pic
	if err == errNoPic {
		err = nil
	}
	return rc, err
}

func LocalResource(prefix, name string) string {
	return fmt.Sprintf("%s?key=%s", prefix, url.QueryEscape(name))
}

func ReplaceResources(item pkg.Item, apiPrefix string) error {
	id := DocId(item)
	err := mapResource(item, func(it pkg.Item) error {
		u := it["url"].(string)
		it["url"] = LocalResource(apiPrefix, u)
		return nil
	})
	if err == errNoPic {
		zap.S().Info("no pic info to replace")
		err = nil
	}
	if err != nil {
		return err
	}

	if HasVideo(item) {
		item = OriginTweet(item)
		item[VideoResourceKey] = LocalResource(apiPrefix, id)
	}

	return nil
}
