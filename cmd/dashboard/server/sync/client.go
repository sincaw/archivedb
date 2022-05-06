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

func (h *httpCli) GetImages(urls []string) (pkg.Resources, error) {
	var (
		rc = make(pkg.Resources)
		mu sync.Mutex
		eg errgroup.Group
	)
	for _, url := range urls {
		url := url
		eg.Go(func() error {
			resp, err := h.Get(url)
			if err != nil {
				return err
			}
			mu.Lock()
			rc[url] = resp
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
func (h httpCli) FetchVideoIfNeeded(item pkg.Item) ([]byte, error) {
	if !utils.HasVideo(item) {
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

	type showResp struct {
		Ok   int `json:"ok"`
		PageInfo struct {
			MediaInfo struct{
				Mp4HdUrl string `json:"mp4_hd_url"`
				Mp4720pMp4 string `json:"mp4_720p_mp4"`
			} `json:"media_info"`
		} `json:"page_info"`
	}

	show := new(showResp)
	err = json.Unmarshal(resp, show)
	if err != nil {
		return nil, fmt.Errorf("unmarshal fail with context %s, err %v", string(resp), err)
	}
	if show.Ok != 1 {
		return nil, fmt.Errorf("wrong response %+v", show)
	}
	url := show.PageInfo.MediaInfo.Mp4720pMp4
	if url == "" {
		url = show.PageInfo.MediaInfo.Mp4HdUrl
	}
	if url == "" {
		return nil, fmt.Errorf("can not find video url %q", string(resp))
	}

	fmt.Printf("fetch video %s\n", url)
	return h.Get(url)
}
