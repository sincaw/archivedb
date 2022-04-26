package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/jialeicui/archivedb/pkg"
)

var (
	db               pkg.DB
	defaultPageLimit = 20
)

// TODO support offset
func ListHandler(w http.ResponseWriter, r *http.Request) {
	limit := defaultPageLimit
	vars := r.URL.Query()
	if v, ok := vars["limit"]; ok {
		l, err := strconv.Atoi(v[0])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%v", err)
			return
		}
		if l < 1 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "wrong limit %d", l)
			return
		}
		limit = l
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
		if count > limit {
			break
		}
		v, err := iter.Value()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%v", err)
			return
		}
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
	db, err = pkg.New(os.Args[1])
	if err != nil {
		panic(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/list", ListHandler)
	r.HandleFunc("/resource", ResourceHandler)
	http.Handle("/", r)
	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:8000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
