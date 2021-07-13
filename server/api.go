package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/DataHenHQ/till/internal/tillclient"
	"github.com/DataHenHQ/till/server/handlers"
	"github.com/gorilla/mux"
)

type APIServer struct {
	server   *http.Server
	port     string
	instance *tillclient.Instance
}

func NewAPIServer(port string, i *tillclient.Instance) (s *APIServer, err error) {

	r := mux.NewRouter()

	r.HandleFunc("/", handlers.IndexHandler)

	if !LoggerConfig.Disabled {
		r.HandleFunc("/requests", handlers.RequestIndexHandler)
		r.HandleFunc("/requests/{rid}", handlers.RequestShowHandler)
		r.HandleFunc("/requests/{rid}/content", handlers.RequestContentShowHandler)
	}

	// wildcard for content, so that URL path is similar to original request
	r.PathPrefix("/requests/{rid}/content/").HandlerFunc(handlers.RequestContentShowHandler)

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
	fmt.Printf("Starting Till Web UI on http://localhost:%v\n", s.port)
	if err := s.server.ListenAndServe(); err != nil {
		log.Println("shutting down DataHen TIll Web UI:", err)
	}
}
