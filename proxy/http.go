package proxy

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/DataHenHQ/tillup/features"
	"github.com/DataHenHQ/tillup/interceptors"
	"github.com/DataHenHQ/tillup/sessions"
	"github.com/DataHenHQ/tillup/sessions/sticky"
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

	// Create a till session
	sess := sessions.New()

	// Generate the Page
	pconf := generatePageConfig(sreq)
	scheme := sreq.URL.Scheme
	p, err := NewPageFromRequest(sreq, scheme, pconf)
	if err != nil {
		return err
	}

	// If Interceptor is allowed and it matches
	if features.Allow(features.Interceptors) {
		if ok, in := interceptors.Matches(sreq); ok && in != nil {
			resp, err := in.CreateResponse()
			if err != nil {
				return err
			}

			writeToSource(sconn, resp, p)

			// Increment reqs and intercepted reqs stats
			incrRequestStatDelta()
			incrInterceptedRequestStatDelta()
			return nil
		}
	}

	// If StickySession is allowed, then set the sticky session
	if features.Allow(features.StickySessions) {
		s, err := sticky.GetSessionFromRequest(sreq, (sessions.PageConfig)(*pconf))
		if err != nil {
			return err
		}
		if s != nil {
			sess = s
		}
	}

	// Send request to target server
	tresp, err := sendToTarget(sreq.Context(), sconn, sreq, scheme, p, pconf, sess)
	if err != nil {
		return err
	}
	defer tresp.Body.Close()

	// Write the request log
	defer writeHarLog()
	defer incrRequestStatDelta()

	// Write response back to the source connection
	writeToSource(sconn, tresp, p)
	return nil
}
