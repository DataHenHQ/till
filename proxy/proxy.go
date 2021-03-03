package proxy

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/DataHenHQ/datahen/pages"
	"github.com/DataHenHQ/useragent"
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

	// ProxyFile points to the path of the txt file that contains a list of proxies
	ProxyFile = ""

	// ProxyURLs are external proxies that will be randomized
	ProxyURLs = []string{}

	// ProxyCount is the total count of proxies used.
	ProxyCount int
)

func init() {
	// loadCAVar()
	// loadCAVarFromFile()
}

func NewPageFromRequest(r *http.Request, scheme string, config *PageConfig) (p *pages.Page, err error) {
	p = new(pages.Page)

	u := r.URL
	u.Host = r.Host
	u.Scheme = scheme
	p.SetURL(u.String())

	p.SetMethod(r.Method)

	// build the page headers
	nh := map[string]interface{}{}
	for name, values := range r.Header {
		nh[name] = strings.Join(values, ",")
	}

	// remove User-Agent header if we force-user agent
	if config.ForceUA {
		delete(nh, "User-Agent")
	}

	// delete any other proxy related header
	delete(nh, "Proxy-Connection")

	// finally set the header
	p.SetHeaders(nh)

	// fetch type will always be "standard" for Till
	p.FetchType = "standard"
	p.UaType = config.UaType

	// set the body
	rBody, _ := ioutil.ReadAll(r.Body)
	p.SetBody(string(rBody))

	// set defaults
	p.SetUaType(config.UaType)
	p.SetFetchType("standard")
	p.SetPageType("default")

	// set the GID
	gid, err := pages.GenerateGID(p)
	if err != nil {
		return nil, err
	}
	p.SetGID(gid)

	return p, nil
}

func sendToTarget(sconn net.Conn, sreq *http.Request, scheme string, config *PageConfig) (tresp *http.Response, err error) {
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
	if config.UseProxy {
		// randomizes the proxy
		u := getRandom(ProxyURLs)
		p, err := url.Parse(u)
		if err != nil {
			return nil, err
		}
		t.Proxy = http.ProxyURL(p)
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

	// Delete headers related to proxy usage
	treq.Header.Del("Proxy-Connection")

	// if ForceUA is true, then override User-Agent header with a random UA
	if ForceUA {
		if err := generateRandomUA(th, UAType); err != nil {
			return nil, err
		}
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
func generateRandomUA(h http.Header, uaType string) (err error) {
	var ua string
	switch uaType {
	case "desktop":
		ua, err = useragent.Desktop()
		if err != nil {
			return err
		}
	case "mobile":
		ua = useragent.Mobile()
	}

	if ua == "" {
		return errors.New(fmt.Sprint("generated empty user agent string for", uaType))
	}

	h.Set("User-Agent", ua)
	return nil
}

func writeToSource(sconn net.Conn, tresp *http.Response, p *pages.Page) (err error) {
	// add X-DH-GID to the response
	if p != nil {
		tresp.Header.Set("X-DH-GID", p.GetGID())
	}

	tresp.Write(sconn)
	return nil
}
