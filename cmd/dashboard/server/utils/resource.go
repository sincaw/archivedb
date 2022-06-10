package utils

import (
	"fmt"

	"github.com/sincaw/archivedb/pkg"
)

const (
	retweet    = "retweeted_status"
	PicInfoKey = "pic_infos"
)

const (
	VideoResourceKey = "video"
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

func ReplaceResources(item pkg.Item, apiPrefix string) {
	id := DocId(item)
	if HasVideo(item) {
		item = OriginTweet(item)
		item[VideoResourceKey] = fmt.Sprintf("%s/%s.mp4", apiPrefix, id)
	}
}
