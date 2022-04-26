package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/jialeicui/archivedb/pkg"
)

func getImage(url string) ([]byte, error) {
	if url == "" {
		url = "https://picsum.photos/id/0/200/300"
	}
	response, e := http.Get(url)
	if e != nil {
		log.Fatal(e)
	}
	defer response.Body.Close()
	return ioutil.ReadAll(response.Body)
}

func main() {
	c, err := ioutil.ReadFile("demo.json")
	if err != nil {
		panic(err)
	}

	item := pkg.Item{}
	err = bson.UnmarshalExtJSON(c, true, &item)
	if err != nil {
		panic(err)
	}

	os.RemoveAll("data")
	d, err := pkg.New("data")
	if err != nil {
		panic(err)
	}
	defer d.Close()

	binaryContent, err := getImage("")
	if err != nil {
		panic(err)
	}

	for _, i := range item["data"].(bson.A) {
		it := i.(bson.M)
		err = d.Put(&it,
			pkg.WithResources(pkg.Resources{"img": binaryContent}),
			pkg.WithKey([]byte(it["idstr"].(string))),
		)
		if err != nil {
			panic(err)
		}
	}
	it, err := d.Find(pkg.Query{})
	if err != nil {
		panic(err)
	}

	defer it.Release()
	for it.Next() {
		it, err := it.Value()
		if err != nil {
			panic(err)
		}
		fmt.Println(it)
	}
}
