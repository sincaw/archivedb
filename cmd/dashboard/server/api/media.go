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

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/sincaw/archivedb/cmd/dashboard/server/common"
	"github.com/sincaw/archivedb/cmd/dashboard/server/utils"
)

var (
	logger = utils.Logger()
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
	l := logger.With("api", "list")
	vars := r.URL.Query()
	limit, err := getIntVal(vars, "limit", defaultPageLimit, 1)
	if err != nil {
		l.Error("parse limit fail ", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%v", err)
		return
	}

	offset, err := getIntVal(vars, "offset", 0, 0)
	if err != nil {
		l.Error("parse offset fail ", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%v", err)
		return
	}

	// order by fav time
	iter, err := a.fav.Range(nil, nil, false)
	if err != nil {
		l.Error("range db fail ", err)
		responseServerError(w, err)
		return
	}
	defer iter.Release()

	items := bson.A{}
	count := 0
	for iter.Next() {
		k, err := iter.Value()
		if err != nil {
			l.Error("get item fail ", err)
			responseServerError(w, err)
			return
		}
		v, err := a.ns.DocBucket().GetDoc(k)
		if err != nil {
			l.Errorf("get doc by key %v fail %v", k, err)
			responseServerError(w, err)
			return
		}

		// TODO count if incorrect when filter enabled
		if a.config.Server.Filter.Ignore(v) {
			continue
		}
		count++
		if count <= offset {
			continue
		}
		if count > (offset + limit) {
			break
		}
		items = append(items, v)
	}

	total, err := a.fav.Count(nil, nil)
	if err != nil {
		l.Error("get total items fail ", err)
		responseServerError(w, err)
		return
	}
	content, err := bson.MarshalExtJSON(bson.M{"data": items, "total": total}, false, true)
	if err != nil {
		l.Error("marshal result fail ", err)
		responseServerError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(content)
	if err != nil {
		l.Error("write content fail: ", err)
	}
}

// VideoHandler handles media resource (binary) fetching
// Only support jpg image and mp4 video for now
func (a *Api) VideoHandler(w http.ResponseWriter, r *http.Request) {
	l := logger.With("api", "video")

	l.With("method", r.Method, "param", r.URL.Query(), "vars", mux.Vars(r)).Debug("query resource api")

	key := mux.Vars(r)["id"]
	if key == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	arr := strings.Split(key, ".")
	key = arr[0]

	meta, err := a.ns.ObjectBucket().GetMeta([]byte(key))
	if err != nil {
		l.Error("get object meta fail ", err)
		responseServerError(w, err)
		return
	}

	headOnly := r.Header.Get("Sec-Fetch-Dest") == "document"

	var (
		re       = regexp.MustCompile("bytes=(\\d+)-(\\d*)")
		rangeStr = r.Header.Get("range")
		match    = re.FindStringSubmatch(rangeStr)

		start = 0
		end   = meta.TotalLen - 1
	)

	if len(match) != 0 {
		if len(match) != 3 {
			responseServerError(w, fmt.Errorf("failed to parse range %q", rangeStr))
			return
		}
		start, err = strconv.Atoi(match[1])
		if err != nil {
			responseServerError(w, fmt.Errorf("failed to parse start %v", err))
			return
		}
		if len(match[2]) > 0 {
			tmp, err := strconv.Atoi(match[2])
			if err != nil {
				responseServerError(w, fmt.Errorf("failed to parse end %v", err))
				return
			}
			end = tmp
		}
	}

	w.Header().Add("Content-Type", meta.Mime)
	w.Header().Add("access-control-allow-methods", "GET")
	w.Header().Add("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, meta.TotalLen))
	w.Header().Add("Content-Length", fmt.Sprintf("%d", meta.TotalLen))
	w.Header().Add("accept-ranges", "bytes")
	if len(match) == 0 {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusPartialContent)
	}

	if !headOnly {
		l.Debugf("partial request %d-%d", start, end)
		buf := make([]byte, end+1-start)
		n, err := a.ns.ObjectBucket().GetAt([]byte(key), buf, start)
		if err != nil {
			l.Error("get partial content fail ", err)
			responseServerError(w, err)
			return
		}
		w.Write(buf[:n])
		l.Debugf("partial response %d bytes", n)
	}
}

func (a *Api) ImageHandler(w http.ResponseWriter, r *http.Request) {
	var (
		l   = logger.With("api", "image")
		key = mux.Vars(r)["id"]
	)

	if key == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	arr := strings.Split(key, ".")
	key = arr[0]
	l = l.With("id", key)

	rc, _, err := a.ns.ObjectBucket().Get([]byte(key))
	if err != nil {
		l.Error("get image fail: ", err)
		responseServerError(w, err)
		return
	}

	w.Header().Add("Content-Type", common.MimeImage)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(rc)
	if err != nil {
		l.Error("write content fail: ", err)
	}
}
