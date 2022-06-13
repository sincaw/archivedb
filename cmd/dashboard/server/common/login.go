package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func extractJson(body []byte) ([]byte, error) {
	p := regexp.MustCompile(".*?({.*}).*")
	match := p.FindSubmatch(body)
	if len(match) != 2 {
		return nil, fmt.Errorf("can not get valid json with resp %q", string(body))
	}

	return match[1], nil
}

func QRCode() (id string, content []byte, err error) {
	tm := time.Now().Unix() * 10000
	url := fmt.Sprintf("https://login.sina.com.cn/sso/qrcode/image?entry=sso&size=180&service_id=pc_protection&callback=STK_%d", tm)

	cli, err := NewWithHeader(map[string]string{"referer": "https://passport.weibo.com/"})
	if err != nil {
		return "", nil, err
	}
	defer cli.CloseIdleConnections()

	resp, err := cli.Get(url)
	if err != nil {
		return "", nil, err
	}

	content, err = extractJson(resp)
	if err != nil {
		return "", nil, err
	}

	type qrResp struct {
		Data struct {
			QRid  string `json:"qrid"`
			Image string `json:"image"`
		} `json:"data"`
	}
	var qr = new(qrResp)
	err = json.Unmarshal(content, qr)
	if err != nil {
		return "", nil, fmt.Errorf("unmarshal qr resp fail with err: %v, content %q", err, string(content))
	}

	content, err = cli.Get(qr.Data.Image)
	if err != nil {
		return "", nil, err
	}

	return qr.Data.QRid, content, nil
}

func CheckScanState(ctx context.Context, id string) (alt string, err error) {
	tm := time.Now().Unix() * 10000
	url := fmt.Sprintf("https://login.sina.com.cn/sso/qrcode/check?entry=sso&qrid=%s&callback=STK_%d", id, tm)

	cli, err := NewWithHeader(map[string]string{"referer": "https://weibo.com/"})
	if err != nil {
		return "", err
	}
	defer cli.CloseIdleConnections()

	type checkResp struct {
		RetCode int `json:"retcode"`
		Data    struct {
			Alt string `json:"alt"`
		} `json:"data"`
	}

	for {
		resp, err := cli.Get(url)
		if err != nil {
			return "", err
		}
		content, err := extractJson(resp)
		if err != nil {
			return "", err
		}
		var check = new(checkResp)
		err = json.Unmarshal(content, check)
		if err != nil {
			return "", fmt.Errorf("unmarshal check resp fail with err: %v, content: %q", err, string(content))
		}
		if check.RetCode == 20000000 {
			return check.Data.Alt, nil
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}

		time.Sleep(time.Second)
	}
}

func GetCookie(alt string) (string, error) {
	url := "https://login.sina.com.cn/sso/login.php"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	qs := map[string]string{
		"entry":       "weibo",
		"returntype":  "TEXT",
		"crossdomain": "1",
		"cdult":       "3",
		"domain":      "weibo.com",
		"alt":         alt,
		"savestate":   "30",
		"callback":    fmt.Sprintf("STK_%d", time.Now().Unix()*10000),
	}
	q := req.URL.Query()
	for k, v := range qs {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	var cookies []string
	for _, c := range resp.Cookies() {
		cookies = append(cookies, fmt.Sprintf("%s=%s", c.Name, c.Value))
	}
	return strings.Join(cookies, "; "), nil
}
