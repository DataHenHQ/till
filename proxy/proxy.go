package proxy

import (
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"golang.org/x/net/publicsuffix"
)

// ProxyURLs is the current configuration of this proxy
var ProxyURLs []string

// HandleHTTP proxies the request from source to target
func HandleHTTP(sw http.ResponseWriter, sreq *http.Request) error {
	// Hijack the source connection
	sconn, _, err := sw.(http.Hijacker).Hijack()
	if err != nil {
		e := errors.New(fmt.Sprint("unable to hijack the source connection", sreq.Host, err))
		return e
	}
	defer sconn.Close()

	// Send request to target server
	tresp, err := sendToTarget(sconn, sreq)
	if err != nil {
		return err
	}
	defer tresp.Body.Close()

	// Write response back to the source connection
	writeToSource(sconn, tresp)
	return nil
}

func sendToTarget(sconn net.Conn, sreq *http.Request) (tresp *http.Response, err error) {
	// create transport for client
	t := &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		DisableCompression:    false,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// set proxy if specified
	if nproxies := len(ProxyURLs); nproxies > 0 {
		// randomizes the proxy
		u := getRandom(ProxyURLs)
		p, err := url.Parse(u)
		if err != nil {
			t.Proxy = http.ProxyURL(p)
		}
	}

	// create cookiejar
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}

	// create target client
	tclient := &http.Client{
		Timeout:   120 * time.Second,
		Transport: t,
		Jar:       jar,
	}

	// create target request
	treq, err := http.NewRequestWithContext(sreq.Context(), sreq.Method, sreq.RequestURI, sreq.Body)
	if err != nil {
		return nil, err
	}

	// send the actual request to target server
	tresp, err = tclient.Do(treq)

	return tresp, err
}

func writeToSource(sconn net.Conn, tresp *http.Response) (err error) {
	tresp.Write(sconn)
	return nil
}

func getRandom(s []string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	i := r.Intn(len(s))

	return s[i]
}
