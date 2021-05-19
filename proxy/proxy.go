package proxy

import (
	"bytes"
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
	"github.com/DataHenHQ/till/internal/tillclient"
	"github.com/DataHenHQ/tillup/sessions"
	"github.com/DataHenHQ/tillup/sessions/sticky"
	"github.com/DataHenHQ/useragent"
	"github.com/google/martian/v3/har"
	"golang.org/x/net/publicsuffix"
)

var (
	// Token is the Till auth token
	Token string

	// Instance is the name of this till instance
	Instance string

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

	harlogger = har.NewLogger()

	// ReleaseVersion is the version of Till release
	ReleaseVersion = "dev"

	StatMu *tillclient.InstanceStatMutex
)

func init() {

	// init har logger
	harlogger.Export().Log.Creator.Name = "DataHen Till"
	harlogger.Export().Log.Creator.Version = "dev"
	harlogger.SetOption(har.PostDataLogging(true))
	harlogger.SetOption(har.BodyLogging(false))
}

func NewPageFromRequest(r *http.Request, scheme string, pconf *PageConfig) (p *pages.Page, err error) {
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
	if pconf.ForceUA {
		delete(nh, "User-Agent")
	}

	// delete any other proxy related header
	delete(nh, "Proxy-Connection")

	// finally set the header
	p.SetHeaders(nh)

	// fetch type will always be "standard" for Till
	p.FetchType = "standard"
	p.UaType = pconf.UaType

	// read the request body, save it and set it back to the request body
	rBody, _ := ioutil.ReadAll(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewReader(rBody))
	p.SetBody(string(rBody))

	// set defaults
	p.SetUaType(pconf.UaType)
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

func sendToTarget(sconn net.Conn, sreq *http.Request, scheme string, p *pages.Page, pconf *PageConfig, sess *sessions.Session) (tresp *http.Response, err error) {
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
		MaxIdleConns:          1,
		MaxIdleConnsPerHost:   1,
		IdleConnTimeout:       1 * time.Millisecond,
		MaxConnsPerHost:       1,
	}
	defer t.CloseIdleConnections()

	// set proxy if specified
	if pconf.UseProxy {

		// using till session's proxy URL, or generate random proxy
		u := sess.ProxyURL
		if u == "" {
			u = getRandom(ProxyURLs)
		}

		// set the proxy
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

	// copy the body as *bytes.Reader to properly set the treq's body and content-length
	srBody, _ := ioutil.ReadAll(sreq.Body)
	sreq.Body = ioutil.NopCloser(bytes.NewReader(srBody))
	p.SetBody(string(srBody))

	// create target request
	treq, err := http.NewRequestWithContext(sreq.Context(), sreq.Method, sreq.RequestURI, bytes.NewReader(srBody))
	if err != nil {
		return nil, err
	}
	// build the target request
	u := sreq.URL
	u.Host = sreq.Host
	u.Scheme = scheme
	treq.URL = u
	treq.Host = u.Host

	// if there are cookies on the session, set it in the cookiejar
	if len(sess.Cookies) > 0 {
		if pconf.StickyCookies {
			tclient.Jar.SetCookies(treq.URL, sess.Cookies)
		}
	}

	// copy source headers into target headers
	th := copySourceHeaders(sreq.Header)
	if th != nil {
		treq.Header = th
	}

	// Delete headers related to proxy usage
	treq.Header.Del("Proxy-Connection")

	// if ForceUA is true, then override User-Agent header with a random UA
	if ForceUA {

		// using till session's user agent, or generate random one
		ua := sess.UA
		if ua == "" {
			ua, err = generateRandomUA(UAType)
			if err != nil {
				return nil, err
			}
		}

		// Set the ua on the target header
		th.Set("User-Agent", ua)
	}

	// record request to HAR
	if err := harlogger.RecordRequest(p.GetGID(), treq); err != nil {
		return nil, err
	}

	// send the actual request to target server
	tresp, err = tclient.Do(treq)
	if err != nil {
		return nil, err
	}

	// save the cookies from cookiejar to the session
	if !sess.IsZero() {
		if pconf.StickyCookies {
			sess.Cookies = tclient.Jar.Cookies(treq.URL)
		}
		sticky.SaveSession(sess)
	}

	// record response to HAR
	if err := harlogger.RecordResponse(p.GetGID(), tresp); err != nil {
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
func generateRandomUA(uaType string) (ua string, err error) {
	switch uaType {
	case "desktop":
		ua, err = useragent.Desktop()
		if err != nil {
			return "", err
		}
	case "mobile":
		ua = useragent.Mobile()
	}

	if ua == "" {
		return "", errors.New(fmt.Sprint("generated empty user agent string for", uaType))
	}

	return ua, nil
}

func writeToSource(sconn net.Conn, tresp *http.Response, p *pages.Page) (err error) {
	// add X-DH-GID to the response
	if p != nil {
		tresp.Header.Set("X-DH-GID", p.GetGID())
	}

	tresp.Write(sconn)

	return nil
}

// Atomically increments request delta in the instance stat
func incrRequestStatDelta() {
	StatMu.Mutex.Lock()

	// increment the requests counter
	*(StatMu.InstanceStat.Requests) = *(StatMu.InstanceStat.Requests) + uint64(1)
	StatMu.Mutex.Unlock()

}
