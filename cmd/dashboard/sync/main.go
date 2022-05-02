package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/jialeicui/archivedb/cmd/dashboard/utils"
	"github.com/jialeicui/archivedb/pkg"
)

type Config struct {
	Uid    string `yaml:"uid"`
	Cookie string `yaml:"cookie"`
}

const (
	configPath = ".config.yaml"
	dbPath     = ".data"
)

var (
	flagStartPage = flag.Int("p", 1, "start page")
	flagSyncMode = flag.Bool("ar", false, "archive mode, auto stop when we meet exists key")
)

func main() {
	flag.Parse()

	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	config := new(Config)
	err = yaml.Unmarshal(content, config)
	if err != nil {
		panic(err)
	}

	db, err := pkg.New(dbPath, false)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	cli, err := NewWithHeader(map[string]string{"cookie": config.Cookie})
	if err != nil {
		panic(err)
	}

	page := *flagStartPage

mainLoop:
	for {
		fmt.Printf("page %d\n", page)
		url := fmt.Sprintf("https://weibo.com/ajax/favorites/all_fav?uid=%s&page=%d", config.Uid, page)
		page++
		resp, err := cli.Get(url)
		if err != nil {
			panic(err)
		}

		item := pkg.Item{}
		err = bson.UnmarshalExtJSON(resp, true, &item)
		if err != nil {
			panic(err)
		}
		ok := item["ok"]
		if ok != int32(1) {
			panic(fmt.Sprintf("invalid content: %q", string(resp)))
		}

		items := item["data"].(bson.A)
		if len(items) == 0 {
			break mainLoop
		}
		for _, i := range items {
			it := i.(bson.M)
			key := []byte(utils.DocId(it))
			yes, err := db.Exists(key)
			if err != nil {
				zap.S().Error(err)
				return
			}
			if yes {
				if *flagSyncMode {
					break mainLoop
				}
				zap.S().Infof("skip with key %q", string(key))
				continue
			}
			urls, err := utils.FilterResources(it)
			if err != nil {
				panic(err)
			}
			resources, err := cli.GetImages(urls)
			if err != nil {
				panic(err)
			}

			err = cli.FetchLongTextIfNeeded(it)
			if err != nil {
				zap.S().Error(err)
			}
			err = db.Put(&it,
				pkg.WithKey(key),
				pkg.WithResources(resources),
			)
			if err != nil {
				panic(err)
			}
		}
	}
	db.Compact()
	fmt.Println("done")
}
