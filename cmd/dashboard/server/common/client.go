package common

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

var (
	ErrStatus = fmt.Errorf("error status code")
)

type HttpCli struct {
	http.Client

	header http.Header
}

func NewWithHeader(header map[string]string) (cli *HttpCli, err error) {
	h := http.Header{}
	for k, v := range header {
		h.Set(k, v)
	}

	cli = &HttpCli{
		Client: http.Client{},
		header: h,
	}
	return
}

func (h *HttpCli) Post(url string, data string) ([]byte, error) {
	return h.do("POST", url, []byte(data))
}

func (h *HttpCli) do(method, url string, data []byte) ([]byte, error) {
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
	if resp.StatusCode != http.StatusOK {
		return nil, ErrStatus
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (h *HttpCli) Get(url string) ([]byte, error) {
	return h.do("GET", url, nil)
}

type CookieAcceptor interface {
	Accept(cookie string)
}
