package api

import (
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
	ns     pkg.Namespace
	config common.WebServerConfig
}

// New Api instance using and db ns config
func New(ns pkg.Namespace, config common.WebServerConfig) *Api {
	return &Api{
		ns:     ns,
		config: config,
	}
}

// Serve http server
func (a *Api) Serve() error {
	r := mux.NewRouter()
	r.HandleFunc(uriDocList, a.ListHandler)
	r.HandleFunc(uriResource, a.ResourceHandler)
	r.HandleFunc(uriDocUpdateSettings, a.ResourceHandler).Methods("POST", "PATCH")

	handler := AssetHandler("/", "build")
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
