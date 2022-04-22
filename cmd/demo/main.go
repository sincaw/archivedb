package main

import (
	"fmt"

	"github.com/jialeicui/archivedb/pkg"
	"github.com/jialeicui/archivedb/pkg/local"
)

func main() {
	d, err := local.New("data")
	if err != nil {
		panic(err)
	}
	defer d.Close()

	err = d.Put(&pkg.Item{"A": "B", "C": pkg.Item{"D": "E"}})
	if err != nil {
		panic(err)
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
