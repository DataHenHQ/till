package server

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DataHenHQ/till/internal/tillclient"
	"github.com/DataHenHQ/till/proxy"
	"github.com/DataHenHQ/till/server/handlers"
	"github.com/DataHenHQ/tillup/cache"
	"github.com/DataHenHQ/tillup/interceptors"
	"github.com/DataHenHQ/tillup/logger"
	"github.com/DataHenHQ/tillup/sessions"

	"github.com/DataHenHQ/tillup"
)

var (
	Token          string
	InstanceName   string
	StatMu         tillclient.InstanceStatMutex
	ProxyURLs      = []string{}
	ProxyCount     = 0
	DBPath         string
	Interceptors   []interceptors.Interceptor
	CacheConfig    cache.Config
	LoggerConfig   logger.Config
	SessionsConfig sessions.Config

	// current instance from the server
	curri tillclient.Instance

	// content holds our static web server content.
	//go:embed templates/*
	embeddedTemplates embed.FS
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

	i, _, err = client.Instances.Get(context.Background(), InstanceName)
	if err != nil {
		if errors.Is(err, tillclient.ErrNotFound) {
			log.Fatalf("Instance with the name '%v' is not found. Please create the instance at https://till.datahen.com/instances\n", InstanceName)
		} else {
			log.Fatal(err)
		}
		log.Fatal(err)
	}

	// set the current instance global var
	curri = *i

	// Set the features, etc for this instance
	if err := tillup.Init(i.GetFeatures(), ProxyURLs, DBPath, Interceptors, SessionsConfig, CacheConfig, LoggerConfig); err != nil {
		log.Fatal(err)
	}

	return true, i
}

// Serve runs the Till server to start accepting the proxy requests
func Serve(port string, uiport string) {
	defer tillup.Close()

	// Pass necessary vars to the handlers
	handlers.SetEmbeddedTemplates(&embeddedTemplates)
	handlers.InstanceName = InstanceName
	handlers.CurrentInstance = &curri
	handlers.StatMu = &StatMu
	handlers.LoggerConfig = LoggerConfig

	// Wait for os signal to gracefully shutdown the server
	//
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)

	// Validates this instance with the cloud
	ok, i := validateInstance()
	if !ok {
		return
	}

	// Start recurning stats update to cloud
	//
	StatMu = newInstanceStatMutex()
	proxy.StatMu = &StatMu
	go startRecurringStatUpdate()

	// Starts the Proxy server
	//
	prox, err := NewProxyServer(port, i)
	if err != nil {
		log.Fatal("Unable to start Till Proxy Server")
	}
	go prox.ListenAndServe()

	// Starts the UI server
	//
	ui, err := NewUIServer(uiport, i)
	if err != nil {
		log.Fatal("Unable to start Till UI Server")
	}
	go ui.ListenAndServe()

	// waits for quit signal from OS
	<-quit

	// create context for graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown ui server
	if err := ui.server.Shutdown(ctx); err != nil {
		log.Println("error shutting down DataHen TIll UI server:", err)
	}

	// Shuts down proxy server
	if err := prox.server.Shutdown(ctx); err != nil {
		log.Println("error shutting down DataHen TIll server:", err)
	}
}

// Resets the instant stats delta based on what was uploaded
func resetInstanceStatDelta(is tillclient.InstanceStat) {
	// Lock the mutex first, to prevent edits by other concurent processes
	StatMu.Mutex.Lock()

	// resets the delta by decreasing it by the uploaded stat
	//
	*(StatMu.InstanceStat.Requests) = *(StatMu.InstanceStat.Requests) - is.GetRequests()
	*(StatMu.InstanceStat.InterceptedRequests) = *(StatMu.InstanceStat.InterceptedRequests) - is.GetInterceptedRequests()
	*(StatMu.InstanceStat.FailedRequests) = *(StatMu.InstanceStat.FailedRequests) - is.GetFailedRequests()
	*(StatMu.InstanceStat.CacheHits) = *(StatMu.InstanceStat.CacheHits) - is.GetCacheHits()
	*(StatMu.InstanceStat.CacheSets) = *(StatMu.InstanceStat.CacheSets) - is.GetCacheSets()

	// Unlock the mutex
	StatMu.Mutex.Unlock()
}

func newZeroStat() *uint64 {
	i := uint64(0)
	return &i
}
