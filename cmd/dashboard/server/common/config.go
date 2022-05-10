package common

import (
	"fmt"
	"strings"

	"github.com/sincaw/archivedb/cmd/dashboard/server/utils"
	"github.com/sincaw/archivedb/pkg"
)

const (
	ExtraImagesKey = "archiveImages"
)

// Config for dashboard server behavior
type Config struct {
	Syncer SyncerConfig    `yaml:"syncer"`
	Server WebServerConfig `yaml:"server"`

	DatabasePath string `yaml:"databasePath"`
}

// ContentTypes for content fetching behavior
type ContentTypes struct {
	// fetch long tweet if LongText set to true
	LongText bool `yaml:"longText"`
	// fetch thumbnail for web ui if Thumbnail set to true
	Thumbnail bool `yaml:"thumbnail"`
	// images fetching quality, available options: best, large, middle, none
	ImageQuality ImageQuality `yaml:"imageQuality"`
	// videos fetching quality, available options: best, 720p, 360p, none
	VideoQuality VideoQuality `yaml:"videoQuality"`
}

// SyncerConfig for sync task
type SyncerConfig struct {
	// weibo uid
	Uid string `yaml:"uid"`
	// weibo cookie
	Cookie string `yaml:"cookie"`
	// crontab like string, for sync job run schedule
	Cron string `yaml:"cron"`

	// starting page for sync job
	StartPage int `yaml:"startPage"`
	// sync job stops immediately when it meets existing tweet id when IncrementalMode set to true
	IncrementalMode bool `yaml:"incrementalMode"`
	// content fetching behavior
	ContentTypes ContentTypes `yaml:"contentTypes"`
}

// WebServerConfig for api server
type WebServerConfig struct {
	// web serving address (ip:port)
	Addr string `yaml:"addr"`
	// list api filter
	Filter Filter `yaml:"filter"`
}

// Filter configuration for list handler
type Filter struct {
	// list won't return it if the tweet containers word in filter word list
	Word []string `json:"word"`
	// list won't return it if the tweet id in filter id list
	Id []string `json:"id"`
}

type ImageQuality string

const (
	ImageQualityBest   ImageQuality = "best"
	ImageQualityLarge  ImageQuality = "large"
	ImageQualityMiddle ImageQuality = "middle"
	ImageQualityNone   ImageQuality = "none"
)

var (
	validImageQualities = []ImageQuality{ImageQualityBest, ImageQualityLarge, ImageQualityMiddle, ImageQualityNone}
)

// Valid check if it is valid image quality option
func (q ImageQuality) Valid() error {
	for _, i := range validImageQualities {
		if q == i {
			return nil
		}
	}
	return fmt.Errorf("invalid image quality %q", q)
}

/* Get image url, input item structure
{
  "thumbnail": {
	"url": "https://xxxx.jpg",
	"width": 1,
	"height": 2,
  },
  "bmiddle": {},
  "large": {},
  "original": {},
  "largest": {},
  "mw2000": {},
}
*/
func (q ImageQuality) Get(item pkg.Item, withThumb bool) (url, thumbUrl string, err error) {
	if withThumb {
		// bmiddle as thumbnail (thumbnail is too small to display)
		thumbUrl = imageUrlByKey(item, "bmiddle")
	}
	switch q {
	case ImageQualityNone:
		return
	case ImageQualityMiddle:
		url = imageUrlByKey(item, "bmiddle")
	case ImageQualityLarge:
		url = imageUrlByKey(item, "large")
	case ImageQualityBest:
		url = imageUrlByKey(item, "largest")
	default:
		err = fmt.Errorf("unhandled image quality %q", q)
	}
	return
}

func imageUrlByKey(item pkg.Item, key string) string {
	return item[key].(pkg.Item)["url"].(string)
}

type VideoQuality string

const (
	VideoQualityBest VideoQuality = "best"
	VideoQuality720p VideoQuality = "720p"
	VideoQuality360p VideoQuality = "360p"
	VideoQualityNone VideoQuality = "none"
)

var (
	validVideoQualities = []VideoQuality{VideoQualityBest, VideoQuality720p, VideoQuality360p, VideoQualityNone}
)

// Valid check if it is valid video quality option
func (q VideoQuality) Valid() error {
	for _, i := range validVideoQualities {
		if q == i {
			return nil
		}
	}
	return fmt.Errorf("invalid video quality %q", q)
}

// Ignore check if item should be ignored
func (f *Filter) Ignore(item pkg.Item) (yes bool) {
	id := utils.DocId(item)
	for _, i := range f.Id {
		if id == i {
			return true
		}
	}
	if len(f.Word) == 0 {
		return
	}

	filterWord := func(t string) bool {
		for _, w := range f.Word {
			if strings.Contains(t, w) {
				return true
			}
		}
		return false
	}

	if filterWord(utils.TextRaw(item)) {
		return true
	}

	// TODO reduce filter action when item is no retweet
	return filterWord(utils.TextRaw(utils.OriginTweet(item)))
}
