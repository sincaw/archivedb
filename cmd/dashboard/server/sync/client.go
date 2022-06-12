package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/sincaw/archivedb/cmd/dashboard/server/common"
	"github.com/sincaw/archivedb/cmd/dashboard/server/utils"
	"github.com/sincaw/archivedb/pkg"
)

type httpCli struct {
	http.Client

	header http.Header
}

func NewWithHeader(header map[string]string) (cli *httpCli, err error) {
	h := http.Header{}
	for k, v := range header {
		h.Set(k, v)
	}

	cli = &httpCli{
		Client: http.Client{},
		header: h,
	}
	return
}

func (h *httpCli) Post(url string, data string) ([]byte, error) {
	return h.do("POST", url, []byte(data))
}

func (h *httpCli) do(method, url string, data []byte) ([]byte, error) {
	var body io.Reader
	if data != nil {
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header = h.header

	resp, err := h.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (h *httpCli) Get(url string) ([]byte, error) {
	return h.do("GET", url, nil)
}

func (h *httpCli) GetImages(rcs map[string]resource) (pkg.Resources, error) {
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
			resp, err := h.Get(url)
			if err != nil {
				return err
			}
			mu.Lock()
			rc[n] = resp
			mu.Unlock()
			return nil
		})
	}
	return rc, eg.Wait()
}

func (h httpCli) FetchLongTextIfNeeded(item pkg.Item) error {
	item = utils.OriginTweet(item)
	if _, ok := item["continue_tag"]; !ok {
		return nil
	}

	id, ok := item["mblogid"]
	if !ok {
		return fmt.Errorf("no valid mblog id for %q", utils.DocId(item))
	}
	resp, err := h.Get(fmt.Sprintf("https://weibo.com/ajax/statuses/longtext?id=%s", id))
	if err != nil {
		zap.S().Error(err)
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
		zap.S().Error(err)
		return fmt.Errorf("unmarshal fail with context %s, err %v", string(resp), err)
	}
	item["text_raw"] = long.Data.LongTextContent
	return nil
}

// FetchVideoIfNeeded try parse video in doc
// It returns (nil, nil) when there is no video
// or return video content and nil error
func (h httpCli) FetchVideoIfNeeded(item pkg.Item, vq common.VideoQuality) ([]byte, error) {
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

	resp, err := h.Get(fmt.Sprintf("https://weibo.com/ajax/statuses/show?id=%s", id))
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
	return h.Get(url)
}
