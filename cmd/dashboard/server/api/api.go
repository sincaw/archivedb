package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/sincaw/archivedb/cmd/dashboard/server/common"
	"github.com/sincaw/archivedb/pkg"
)

const (
	uriResource          = "/api/resource"
	uriDocList           = "/api/list"
	uriDocUpdateSettings = "/api/settings"

	defaultPageLimit = 20
)

type Api struct {
	ctx    context.Context
	ns     pkg.Namespace
	config *common.Config
}

// New Api instance using and db ns config
func New(ctx context.Context, ns pkg.Namespace, config *common.Config) *Api {
	return &Api{
		ctx:    ctx,
		ns:     ns,
		config: config,
	}
}

// Serve http server
func (a *Api) Serve() error {
	r := mux.NewRouter()
	r.HandleFunc(uriDocList, a.ListHandler)
	r.HandleFunc(uriResource, a.ResourceHandler)
	r.HandleFunc(uriDocUpdateSettings, a.SettingsHandler).Methods("GET", "POST")

	handler := AssetHandler("/", "build")
	r.PathPrefix("/").Handler(handler)

	addr := a.config.Server.Addr
	srv := &http.Server{
		Handler:      r,
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go func() {
		select {
		case <-a.ctx.Done():
			fmt.Println("reloading...")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			err := srv.Shutdown(ctx)
			if err != nil {
				fmt.Printf("shutdown server fail %v\n", err)
			}
		}
	}()
	fmt.Printf("serving on http://%s\n", addr)
	return srv.ListenAndServe()
}
