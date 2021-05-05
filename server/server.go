package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/DataHenHQ/till/internal/tillclient"
	"github.com/DataHenHQ/till/proxy"
)

var Token string
var Instance string

func validateInstance() (ok bool, i *tillclient.Instance) {
	if Token == "" {
		fmt.Println("You need to specify the Till auth token. To get your auth token, sign up for free at https://till.datahen.com")
		return false, nil
	}

	// init the client
	client, err := tillclient.NewClient(Token)
	if err != nil {
		log.Panic(err)
	}

	i, _, err = client.Instances.Get(context.Background(), Instance)
	if err != nil {
		log.Panic(err)
	}

	return true, i
}

// Serve runs the Till server to start accepting the proxy requests
func Serve(port string) {
	ok, i := validateInstance()
	if !ok {
		return
	}
	// Start the server
	server := &http.Server{
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
	}

	fmt.Printf("Starting DataHen Till server. Instance: %v, port: %v\n", i.GetName(), port)
	log.Fatal(server.ListenAndServe())
}
