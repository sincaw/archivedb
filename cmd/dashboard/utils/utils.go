package utils

import (
	"fmt"
	"net/url"

	"github.com/jialeicui/archivedb/pkg"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	retweet     = "retweeted_status"
	picInfoKey  = "pic_infos"
	originalKey = "original"
)

var (
	errNoPic = fmt.Errorf("no pic info")
)

func DocId(item pkg.Item) string {
	item = OriginTweet(item)
	return item["idstr"].(string)
}

func OriginTweet(item pkg.Item) pkg.Item {
	if _, ok := item[retweet]; ok {
		item = item[retweet].(pkg.Item)
	}
	return item
}

func mapResource(item pkg.Item, fn func(it pkg.Item) error) error {
	item = OriginTweet(item)
	it, ok := item[picInfoKey]
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

func ReplaceResources(item pkg.Item, apiPrefix string) error {
	id := DocId(item)
	mapResource(item, func(it pkg.Item) error {
		u := it["url"].(string)
		it["url"] = fmt.Sprintf("%s?key=%s&name=%s", apiPrefix, id, url.QueryEscape(u))
		return nil
	})
	return nil
}
