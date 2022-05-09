package sync

import (
	"fmt"

	"github.com/sincaw/archivedb/pkg"
)

const (
	extraImagesKey = "archiveImages"
)

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

func (q ImageQuality) valid() error {
	for _, i := range validImageQualities {
		if q == i {
			return nil
		}
	}
	return fmt.Errorf("invalid image quality %q", q)
}

/* get image url, input item structure
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
func (q ImageQuality) get(item pkg.Item, withThumb bool) (url, thumbUrl string, err error) {
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

func (q VideoQuality) valid() error {
	for _, i := range validVideoQualities {
		if q == i {
			return nil
		}
	}
	return fmt.Errorf("invalid video quality %q", q)
}

type ContentTypes struct {
	LongText     bool         `yaml:"longText"`
	Thumbnail    bool         `yaml:"thumbnail"`
	ImageQuality ImageQuality `yaml:"imageQuality"`
	VideoQuality VideoQuality `yaml:"videoQuality"`
}

type Config struct {
	Uid    string `yaml:"uid"`
	Cookie string `yaml:"cookie"`
	Cron   string `yaml:"cron"`

	StartPage       int          `yaml:"startPage"`
	IncrementalMode bool         `yaml:"incrementalMode"`
	ContentTypes    ContentTypes `yaml:"contentTypes"`
}
