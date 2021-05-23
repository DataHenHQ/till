package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/DataHenHQ/till/internal/tillclient"
	"github.com/DataHenHQ/till/proxy"
	"github.com/DataHenHQ/tillup/cache"
	"github.com/DataHenHQ/tillup/interceptors"

	"github.com/DataHenHQ/tillup"
)

var (
	Token        string
	Instance     string
	StatMu       tillclient.InstanceStatMutex
	ProxyURLs    = []string{}
	ProxyCount   = 0
	DBPath       string
	Interceptors []interceptors.Interceptor
	Cache        cache.Config
)

func validateInstance() (ok bool, i *tillclient.Instance) {
	if Token == "" {
		fmt.Println("You need to specify the Till auth token. To get your auth token, sign up for free at https://till.datahen.com")
		return false, nil
	}

	// init the client
	client, err := tillclient.NewClient(Token)
	if err != nil {
		log.Fatal(err)
	}

	i, _, err = client.Instances.Get(context.Background(), Instance)
	if err != nil {
		if errors.Is(err, tillclient.ErrNotFound) {
			log.Fatalf("Instance with the name '%v' is not found. Please create the instance at https://till.datahen.com/instances\n", Instance)
		} else {
			log.Fatal(err)
		}
		log.Fatal(err)
	}

	// Set the features, etc for this instance
	if err := tillup.Init(i.GetFeatures(), ProxyURLs, DBPath, Interceptors, Cache); err != nil {
		log.Fatal(err)
	}

	return true, i
}

// Serve runs the Till server to start accepting the proxy requests
func Serve(port string) {
	ok, i := validateInstance()
	if !ok {
		return
	}

	// init the InstanceStat
	// newZeroStat()
	StatMu = tillclient.InstanceStatMutex{
		Mutex: &sync.Mutex{},
		InstanceStat: tillclient.InstanceStat{
			Requests:            newZeroStat(),
			InterceptedRequests: newZeroStat(),
			FailedRequests:      newZeroStat(),
			CacheHits:           newZeroStat(),
			CacheSets:           newZeroStat(),
			Name:                &Instance,
		},
	}
	proxy.StatMu = &StatMu

	// start the loop to update InstanceStat on the cloud
	go func() {
		client, err := tillclient.NewClient(Token)
		if err != nil {
			log.Fatal(err)
		}

		for {
			time.Sleep(time.Minute)

			// Take a snapshot of the state of the instate stat by doing deep copy
			is := StatMu.InstanceStat.DeepCopy()

			// if instance stat is zero then skip this step
			if is.IsZero() {
				continue
			}

			// Update the stat on the cloud
			_, _, err := client.InstanceStats.Update(context.Background(), is)
			if err != nil {
				fmt.Printf("gotten error: %v\n", err)
			}

			resetInstanceStatDelta(is)

		}
	}()

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

// Resets the instant stats delta based on what was uploaded
func resetInstanceStatDelta(is tillclient.InstanceStat) {
	StatMu.Mutex.Lock()
	// resets the delta by decreasing it by the uploaded stat
	*(StatMu.InstanceStat.Requests) = *(StatMu.InstanceStat.Requests) - is.GetRequests()
	*(StatMu.InstanceStat.InterceptedRequests) = *(StatMu.InstanceStat.InterceptedRequests) - is.GetInterceptedRequests()
	*(StatMu.InstanceStat.FailedRequests) = *(StatMu.InstanceStat.FailedRequests) - is.GetFailedRequests()
	*(StatMu.InstanceStat.CacheHits) = *(StatMu.InstanceStat.CacheHits) - is.GetCacheHits()
	*(StatMu.InstanceStat.CacheSets) = *(StatMu.InstanceStat.CacheSets) - is.GetCacheSets()
	StatMu.Mutex.Unlock()
}

func newZeroStat() *uint64 {
	i := uint64(0)
	return &i
}
