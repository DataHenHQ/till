package proxy

import (
	"crypto/tls"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"golang.org/x/net/publicsuffix"
)

var (
	ca       tls.Certificate
	okHeader = []byte("HTTP/1.1 200 OK\r\n\r\n")

	// ForceUA indicates whether to overwrite all incoming user-agent with a random one
	ForceUA = true

	// UAType specifies what kind of user-agent to generate
	UAType = "desktop"

	dhHeadersRe = regexp.MustCompile(`(?i)^X-DH`)
)

func init() {
	// loadCAVar()
	// loadCAVarFromFile()
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

	// copy source headers into target headers
	th := copySourceHeaders(sreq.Header)
	if th != nil {
		treq.Header = th
	}

	// if ForceUA is true, then override User-Agent header with a random UA
	if ForceUA {
		generateRandomUA(th, UAType)
	}

	// send the actual request to target server
	tresp, err = tclient.Do(treq)
	if err != nil {
		return nil, err
	}

	return tresp, err
}

// copy source headers other than those that starts with X-DH* into target headers
func copySourceHeaders(sh http.Header) (th http.Header) {
	th = make(http.Header)

	if sh == nil {
		return nil
	}

	for key, values := range sh {
		if dhHeadersRe.MatchString(key) {
			continue
		}

		for _, val := range values {
			th.Add(key, val)
		}
	}

	return th
}

// Overrides User-Agent header with a random one
func generateRandomUA(h http.Header, uaType string) {
	// TODO: replace with the real randomizer
	h.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.190 Safari/537.36")
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

func createDirIfNotExist(dirpath string) (err error) {
	if _, err := os.Stat(dirpath); os.IsNotExist(err) {
		return os.MkdirAll(dirpath, os.ModeDir|0755)
	}
	return nil
}

// write the full filepath, also creates non existent directory if not exist
func writeFullFilePath(fullpath string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(fullpath)
	err = createDirIfNotExist(dir)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fullpath, data, perm)
}
