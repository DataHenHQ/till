package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/DataHenHQ/till/internal/tillclient"
	"github.com/gorilla/mux"
	"github.com/unrolled/render"
)

var (

	// rend is html/json/xml renderer
	rend = render.New(render.Options{
		Extensions: []string{".tmpl", ".html"},
		FileSystem: &render.EmbedFileSystem{
			FS: embeddedTemplates,
		},
	})
)

type APIServer struct {
	server   *http.Server
	port     string
	instance *tillclient.Instance
}

func NewAPIServer(port string, i *tillclient.Instance) (s *APIServer, err error) {

	r := mux.NewRouter()

	r.HandleFunc("/", indexHandler)

	s = &APIServer{
		server: &http.Server{
			Addr:         fmt.Sprintf(":%v", port),
			ReadTimeout:  1 * time.Minute,
			WriteTimeout: 1 * time.Minute,
			Handler:      r,
		},
		port:     port,
		instance: i,
	}

	return s, nil
}

func (s *APIServer) ListenAndServe() {
	fmt.Printf("Starting DataHen Till API server. Instance: %v, port: %v\n", s.instance.GetName(), s.port)
	if err := s.server.ListenAndServe(); err != nil {
		log.Println("shutting down DataHen TIll API server:", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	rend.HTML(w, http.StatusOK, "index", map[string]interface{}{
		"Instance":            Instance,
		"Requests":            curri.GetRequests() + int64(*StatMu.Requests),
		"InterceptedRequests": curri.GetInterceptedRequests() + int64(*StatMu.InterceptedRequests),
		"FailedRequests":      curri.GetFailedRequests() + int64(*StatMu.FailedRequests),
		"CacheHits":           curri.GetCacheHits() + int64(*StatMu.CacheHits),
		"CacheSets":           curri.GetCacheSets() + int64(*StatMu.CacheSets),
	})
}
