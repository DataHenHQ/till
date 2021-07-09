package proxy

import (
	"bufio"
	"crypto/tls"
	"log"
	"net"
	"net/http"

	"github.com/DataHenHQ/tillup/features"
	"github.com/DataHenHQ/tillup/interceptors"
	"github.com/DataHenHQ/tillup/sessions"
	"github.com/DataHenHQ/tillup/sessions/sticky"
)

func HandleTunneling(sw http.ResponseWriter, sreq *http.Request) error {

	// get the hostname based on the source request's target host
	hostname := dnsName(sreq.Host)
	if hostname == "" {
		log.Println("cannot determine cert name for " + sreq.Host)
		http.Error(sw, "cannot determine cert name for "+sreq.Host, 503)
		return nil
	}

	// Generate provisional cert to be used to respond to the source request, by pretending to be target certificate
	provisionalCert, err := GenCert([]string{hostname})
	if err != nil {
		log.Println("cert", err)
		http.Error(sw, "no upstream", 503)
		return nil
	}
	sConfig := tls.Config{
		MinVersion: tls.VersionTLS12,
		//CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA},
		Certificates: []tls.Certificate{*provisionalCert},
	}

	// Does TLS Handshake to the source connection
	// tlsHandshakeSource(sconn, provisionalCert)
	sconn, err := Handshake(sw, &sConfig)
	if err != nil {
		return err
	}
	defer sconn.Close()

	// Reads the source's connection into a new request
	reader := bufio.NewReader(sconn)
	treq, err := http.ReadRequest(reader)
	if err != nil {
		log.Println(err)
	}

	// Create a till session
	sess := sessions.New()

	// Generate the PageConfig
	pconf := generatePageConfig(treq)
	scheme := "https"

	// create new page from request
	p, err := NewPageFromRequest(treq, scheme, pconf)
	if err != nil {
		return err
	}

	// If Interceptor is allowed and it matches
	if features.Allow(features.Interceptors) {
		if ok, in := interceptors.Matches(treq); ok && in != nil {
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
		s, err := sticky.GetSessionFromRequest(treq, (sessions.PageConfig)(*pconf))
		if err != nil {
			return err
		}
		if s != nil {
			sess = s
		}
	}

	// Send request to target server
	tresp, err := sendToTarget(sreq.Context(), sconn, treq, scheme, p, pconf, sess)
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

// Handshake hijacks w's underlying net.Conn, responds to the CONNECT request
// and manually performs the TLS handshake. It returns the net.Conn or and
// error if any.
func Handshake(w http.ResponseWriter, config *tls.Config) (net.Conn, error) {
	raw, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		http.Error(w, "no upstream", 503)
		return nil, err
	}
	if _, err = raw.Write(okHeader); err != nil {
		raw.Close()
		return nil, err
	}
	conn := tls.Server(raw, config)
	err = conn.Handshake()
	if err != nil {
		conn.Close()
		raw.Close()
		return nil, err
	}
	return conn, nil
}
