package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"golang.org/x/net/publicsuffix"
)

var (
	ca       tls.Certificate
	okHeader = []byte("HTTP/1.1 200 OK\r\n\r\n")
)

func init() {
	loadCAVar()
}

// Generates geenrates the certificate authority var
func loadCAVar() {
	certPEM, keyPEM, err := GenCA("till")
	if err != nil {
		log.Fatalln("Unable to generate CA", err)
	}
	ca, err = tls.X509KeyPair(certPEM, keyPEM)
	if err == nil {
		ca.Leaf, err = x509.ParseCertificate(ca.Certificate[0])
	}
	return

}

// ProxyURLs is the current configuration of this proxy
var ProxyURLs []string

func sendToTarget(sconn net.Conn, sreq *http.Request, scheme string) (tresp *http.Response, err error) {
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
	// build the target request
	u := sreq.URL
	u.Host = sreq.Host
	u.Scheme = scheme
	treq.URL = u
	treq.Host = u.Host

	// send the actual request to target server
	tresp, err = tclient.Do(treq)
	if err != nil {
		return nil, err
	}

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

// dnsName returns the DNS name in addr, if any.
func dnsName(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return ""
	}
	return host
}
