package server

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/DataHenHQ/till/internal/tillclient"
	"github.com/DataHenHQ/till/proxy"
)

type ProxyServer struct {
	server   *http.Server
	port     string
	instance *tillclient.Instance
}

func NewProxyServer(port string, i *tillclient.Instance) (s *ProxyServer, err error) {
	s = &ProxyServer{
		server: &http.Server{

			Addr:         fmt.Sprintf(":%v", port),
			ReadTimeout:  1 * time.Minute,
			WriteTimeout: 1 * time.Minute,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodConnect {
					proxy.HandleTunneling(w, r)
				} else {
					proxy.HandleHTTP(w, r)
				}
			}),
			// Disable HTTP/2.
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		},
		port:     port,
		instance: i,
	}

	return s, nil
}

func (s *ProxyServer) ListenAndServe() {
	fmt.Printf("Starting Till Proxy server. Instance: %v, port: %v\n", s.instance.GetName(), s.port)
	if err := s.server.ListenAndServe(); err != nil {
		log.Println("shutting down TIll Proxy server:", err)
	}

}
