package api

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/sincaw/archivedb/cmd/dashboard/server/utils"
	"github.com/sincaw/archivedb/pkg"
)

//go:embed build
var Assets embed.FS

type fsFunc func(name string) (fs.File, error)

// Open for fs.FS implementation
func (f fsFunc) Open(name string) (fs.File, error) {
	return f(name)
}

// AssetHandler handles static files such as index.html, *.css, *.js etc.
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

// ListHandler handles tweet list call
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
		utils.ReplaceResources(v, uriResource)
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

// ResourceHandler handles media resource (binary) fetching
// Only support jpg image and mp4 video for now
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
