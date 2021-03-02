package proxy

import (
	"errors"
	"fmt"
	"net/http"
)

// HandleHTTP proxies the request from source to target
func HandleHTTP(sw http.ResponseWriter, sreq *http.Request) error {
	// Hijack the source connection
	sconn, _, err := sw.(http.Hijacker).Hijack()
	if err != nil {
		http.Error(sw, "no upstream", 503)
		e := errors.New(fmt.Sprint("unable to hijack the source connection", sreq.Host, err))
		return e
	}
	defer sconn.Close()

	// Generate the Page
	pconf := generatePageConfig()
	scheme := sreq.URL.Scheme
	p, err := NewPageFromRequest(sreq, scheme, pconf)
	if err != nil {
		return err
	}

	// Send request to target server
	tresp, err := sendToTarget(sconn, sreq, scheme)
	if err != nil {
		return err
	}
	defer tresp.Body.Close()

	// Write response back to the source connection
	writeToSource(sconn, tresp, p)
	return nil
}
