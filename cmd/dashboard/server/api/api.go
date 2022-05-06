package api

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/sincaw/archivedb/cmd/dashboard/server/utils"
	"github.com/sincaw/archivedb/pkg"
)

const (
	resourceApiPrefix = "/resource"
	defaultPageLimit  = 20
)

//go:embed build
var Assets embed.FS

type Config struct {
	Addr   string `yaml:"addr"`
	Filter Filter `yaml:"filter"`
}

type Filter struct {
	Word []string `json:"word"`
	Id   []string `json:"id"`
}

type Api struct {
	ns     pkg.Namespace
	config Config
}

func New(ns pkg.Namespace, config Config) *Api {
	return &Api{
		ns:     ns,
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

	iter, err := a.ns.DocBucket().Find(pkg.Query{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	defer iter.Release()

	items := bson.A{}
	count := 0
	for iter.Next() {
		v, err := iter.ValueDoc()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%v", err)
			return
		}
		if a.config.Filter.Ignore(v) {
			continue
		}
		count++
		if count <= offset {
			continue
		}
		if count > (offset + limit) {
			break
		}
		utils.ReplaceResources(v, resourceApiPrefix)
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
	key := vars["key"][0]
	isVideo := !strings.HasPrefix(key, "http")
	rc, err := a.ns.ObjectBucket().Get([]byte(key))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	// TODO refine partial content parse and response
	if isVideo {
		// request range
		var (
			re       = regexp.MustCompile("bytes=(\\d+)-(\\d*)")
			rangeStr = r.Header.Get("range")
			match    = re.FindStringSubmatch(rangeStr)

			start = 0
			end   = len(rc) - 1
		)

		if len(match) != 0 {
			if len(match) != 3 {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "failed to parse range %q", rangeStr)
				return
			}
			start, err = strconv.Atoi(match[1])
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "failed to parse start %v", err)
				return
			}
			if len(match[2]) > 0 {
				tmp, err := strconv.Atoi(match[2])
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, "failed to parse end %v", err)
					return
				}
				end = tmp
			}
		}

		w.Header().Add("Content-Type", "video/mp4")
		w.Header().Add("access-control-allow-methods", "GET")
		w.Header().Add("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(rc)))
		w.Header().Add("Content-Length", fmt.Sprintf("%d", end-start+1))
		w.Header().Add("accept-ranges", "bytes")
		w.WriteHeader(http.StatusPartialContent)
		w.Write(rc[start : end+1])
	} else {
		w.Header().Add("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(rc)
	}
}

func (f *Filter) Ignore(item pkg.Item) (yes bool) {
	id := utils.DocId(item)
	for _, i := range f.Id {
		if id == i {
			return true
		}
	}
	if len(f.Word) == 0 {
		return
	}

	filterWord := func(t string) bool {
		for _, w := range f.Word {
			if strings.Contains(t, w) {
				return true
			}
		}
		return false
	}

	if filterWord(utils.TextRaw(item)) {
		return true
	}

	// TODO reduce filter action when item is no retweet
	return filterWord(utils.TextRaw(utils.OriginTweet(item)))
}
