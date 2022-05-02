package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/jialeicui/archivedb/cmd/dashboard/utils"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/jialeicui/archivedb/pkg"
)

var (
	db               pkg.DB
	defaultPageLimit = 20
)

const (
	resourceApiPrefix = "/resource"
)

func getIntVal(vars url.Values, key string, defaultVal, minVal int) (int, error) {
	if v, ok := vars[key]; ok {
		l, err := strconv.Atoi(v[0])
		if err != nil {
			return 0, err

		}
		if l < minVal {
			return 0, fmt.Errorf("wrong limit %d", l)
		}
		return l, nil
	}
	return defaultVal, nil
}

// TODO support offset
func ListHandler(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	limit, err := getIntVal(vars, "limit", defaultPageLimit, 1)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%v", err)
		return
	}

	offset, err := getIntVal(vars, "offset", 0, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%v", err)
		return
	}

	iter, err := db.Find(pkg.Query{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	defer iter.Release()

	items := bson.A{}
	count := 0
	for iter.Next() {
		count++
		if count <= offset {
			continue
		}
		if count > (offset + limit) {
			break
		}
		v, err := iter.Value()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%v", err)
			return
		}
		utils.ReplaceResources(*v, resourceApiPrefix)
		items = append(items, v)
	}

	content, err := bson.MarshalExtJSON(bson.M{"data": items}, false, true)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

func ResourceHandler(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	rc, err := db.GetResource([]byte(vars["key"][0]), vars["name"][0])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	w.Header().Add("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusOK)
	w.Write(rc)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("%s <db path>\n", os.Args[0])
		return
	}

	var err error
	db, err = pkg.New(os.Args[1], true)
	if err != nil {
		panic(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/list", ListHandler)
	r.HandleFunc(resourceApiPrefix, ResourceHandler)
	http.Handle("/", r)
	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:8000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
