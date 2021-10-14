package proxy

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/DataHenHQ/tillup/features"
	"github.com/DataHenHQ/tillup/interceptions"
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
	pconf := generatePageConfig(sreq)
	scheme := sreq.URL.Scheme
	p, err := NewPageFromRequest(sreq, scheme, pconf)
	if err != nil {
		return err
	}

	// If Interception is allowed and it matches
	if features.Allow(features.Interceptions) {
		if ok, in := interceptions.Matches(sreq); ok && in != nil {
			resp, err := in.CreateResponse()
			if err != nil {
				return err
			}

			writeToSource(sconn, resp, p)

			// Increment intercepted reqs stats
			incrInterceptedRequestStatDelta()
			return nil
		}
	}

	// Send request to target server
	tresp, err := sendToTarget(sreq.Context(), sconn, sreq, scheme, p, pconf)
	if err != nil {
		return err
	}
	defer tresp.Body.Close()

	// Write response back to the source connection
	writeToSource(sconn, tresp, p)
	return nil
}
