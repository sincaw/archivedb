package api

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/jialeicui/archivedb/cmd/dashboard/server/utils"
	"github.com/jialeicui/archivedb/pkg"
)

const (
	resourceApiPrefix = "/resource"
	defaultPageLimit  = 20
)

//go:embed build
var Assets embed.FS

type Config struct {
	Addr string `yaml:"addr"`
}

type Api struct {
	db     pkg.DB
	config Config
}

func New(db pkg.DB, config Config) *Api {
	return &Api{
		db:     db,
		config: config,
	}
}

type fsFunc func(name string) (fs.File, error)

func (f fsFunc) Open(name string) (fs.File, error) {
	return f(name)
}

func AssetHandler(prefix, root string) http.Handler {
	handler := fsFunc(func(name string) (fs.File, error) {
		assetPath := path.Join(root, name)
		f, err := Assets.Open(assetPath)
		if os.IsNotExist(err) {
			return Assets.Open("build/index.html")
		}
		return f, err
	})
	return http.StripPrefix(prefix, http.FileServer(http.FS(handler)))
}

func (a *Api) Serve() error {
	handler := AssetHandler("/", "build")

	r := mux.NewRouter()
	r.HandleFunc("/list", a.ListHandler)
	r.HandleFunc(resourceApiPrefix, a.ResourceHandler)
	r.PathPrefix("/").Handler(handler)
	http.Handle("/", r)
	srv := &http.Server{
		Handler:      r,
		Addr:         a.config.Addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	fmt.Printf("serving on http://%s\n", a.config.Addr)
	return srv.ListenAndServe()
}

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

func (a *Api) ListHandler(w http.ResponseWriter, r *http.Request) {
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

	iter, err := a.db.Find(pkg.Query{})
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

func (a *Api) ResourceHandler(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	rc, err := a.db.GetResource([]byte(vars["key"][0]), vars["name"][0])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	w.Header().Add("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusOK)
	w.Write(rc)
}
