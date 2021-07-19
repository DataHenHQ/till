package proxy

import (
	"bytes"
	"context"
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
	"github.com/DataHenHQ/tillup/cache"
	"github.com/DataHenHQ/tillup/features"
	"github.com/DataHenHQ/tillup/logger"
	"github.com/DataHenHQ/tillup/sessions"
	"github.com/DataHenHQ/useragent"
	"golang.org/x/net/publicsuffix"
)

var (
	// Token is the Till auth token
	Token string

	// InstanceName is the name of this till instance
	InstanceName string

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

	// ReleaseVersion is the version of Till release
	ReleaseVersion = "dev"

	StatMu *tillclient.InstanceStatMutex

	// Cache is the cache specific config
	CacheConfig cache.Config

	// LoggerConfig is the logger specific config
	LoggerConfig logger.Config

	// SessionsConfig is the sessions specific config
	SessionsConfig sessions.Config
)

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

func logReqSummary(gid, method, url string, respStatus int, cachehit bool) {
	cacheType := "MISS"
	if cachehit {
		cacheType = "HIT "
	}
	fmt.Println(cacheType, gid, method, url, respStatus)
}

func sendToTarget(ctx context.Context, sconn net.Conn, sreq *http.Request, scheme string, p *pages.Page, pconf *PageConfig) (tresp *http.Response, err error) {
	var sess *sessions.Session

	if features.Allow(features.Cache) && !CacheConfig.Disabled {

		// check if past response exist in the cache. if so, then return it.
		cresp, err := cache.GetResponse(ctx, p.GID, pconf.CacheFreshness, pconf.CacheServeFailures)
		if err != nil {
			return nil, err
		}
		// if cachehit ten return the cached response
		if cresp != nil {
			// Increment the CacheHits stats
			incrCacheHitStatDelta()

			logReqSummary(p.GID, sreq.Method, sreq.URL.String(), cresp.StatusCode, true)

			// Build the target req and resp specifically for logging.
			_, treq, terr := buildTargetRequest(scheme, sreq, pconf, sess, p)
			// defer treq.Body.Close()
			if terr == nil && treq != nil {
				// record request and response to the logger
				_, tlerr := logger.StoreItem(ctx, p.GID, treq, cresp, time.Now(), true, (sessions.PageConfig)(*pconf), sess)
				if tlerr != nil {
					return nil, tlerr
				}

			}

			return cresp, nil
		}

	}

	// If StickySession is allowed, then set the sticky session
	if features.Allow(features.StickySessions) && pconf.SessionID != "" {

		// get a session, or a create a new one if it doesn't exist yet.
		sess, err = sessions.GetOrCreateStickySession(ctx, pconf.SessionID, (sessions.PageConfig)(*pconf))
		if err != nil {
			return nil, err
		}

	}

	// build the target request from the source request
	tclient, treq, err := buildTargetRequest(scheme, sreq, pconf, sess, p)
	if err != nil {
		return nil, err
	}

	// record request now, and the logger.Response will be set later once the response comes back.
	rid, tlerr := logger.StoreItem(ctx, p.GID, treq, nil, time.Now(), false, (sessions.PageConfig)(*pconf), sess)
	if tlerr != nil {
		return nil, tlerr
	}

	// send the actual request to target server
	tresp, err = tclient.Do(treq)
	if err != nil {
		return nil, err
	}

	if !sessions.IsSuccess(tresp.StatusCode) {
		incrFailedRequestStatDelta()
	}

	// save the cookies from cookiejar to the session
	if sess != nil && !sess.IsZero() {
		if pconf.StickyCookies {
			if sess.Cookies == nil {
				sess.Cookies = sessions.CookieMap{}
			}
			sess.Cookies.Set(treq.URL, tclient.Jar.Cookies(treq.URL))
		}
		sessions.SaveSession(ctx, sess)
	}

	if features.Allow(features.Cache) && !CacheConfig.Disabled {
		// Store the response to cache
		err := cache.StoreResponse(ctx, p.GID, tresp, nil)
		if err != nil {
			return nil, err
		}

		// Increment the CacheSets stats
		incrCacheSetStatDelta()

	}

	// log the request summary
	logReqSummary(p.GID, sreq.Method, sreq.URL.String(), tresp.StatusCode, false)

	// update response on the logger
	tlerr = logger.UpdateItemResponse(ctx, rid, tresp, sess)
	if tlerr != nil {
		return nil, tlerr
	}

	return tresp, err
}

// buildTargetRequest builds a target request from source request, and etc.
func buildTargetRequest(scheme string, sreq *http.Request, pconf *PageConfig, sess *sessions.Session, p *pages.Page) (*http.Client, *http.Request, error) {
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
		var u string
		if sess != nil {
			u = sess.ProxyURL
		}
		if u == "" {
			u = getRandom(ProxyURLs)
		}

		// set the proxy
		p, err := url.Parse(u)
		if err != nil {
			return nil, nil, err
		}
		t.Proxy = http.ProxyURL(p)
	}

	// create cookiejar
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	// build the target request
	u := sreq.URL
	u.Host = sreq.Host
	u.Scheme = scheme
	treq.URL = u
	treq.Host = u.Host

	// if there are cookies on the session, set it in the cookiejar
	if sess != nil && len(sess.Cookies) > 0 {
		if pconf.StickyCookies {
			tclient.Jar.SetCookies(treq.URL, sess.Cookies.Get(u))
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
		var ua string
		if sess != nil {
			ua = sess.UserAgent
		}
		if ua == "" {
			ua, err = generateRandomUA(UAType)
			if err != nil {
				return nil, nil, err
			}
		}

		// Set the ua on the target header
		th.Set("User-Agent", ua)
	}

	return tclient, treq, nil

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

// Atomically increments intercepted request delta in the instance stat
func incrInterceptedRequestStatDelta() {
	StatMu.Mutex.Lock()

	// increment the requests counter
	*(StatMu.InstanceStat.InterceptedRequests) = *(StatMu.InstanceStat.InterceptedRequests) + uint64(1)
	StatMu.Mutex.Unlock()

}

// Atomically increments failed request delta in the instance stat
func incrFailedRequestStatDelta() {
	StatMu.Mutex.Lock()

	// increment the requests counter
	*(StatMu.InstanceStat.FailedRequests) = *(StatMu.InstanceStat.FailedRequests) + uint64(1)
	StatMu.Mutex.Unlock()

}

// Atomically increments request delta in the instance stat
func incrCacheHitStatDelta() {
	StatMu.Mutex.Lock()

	// increment the CacheHits counter
	*(StatMu.InstanceStat.CacheHits) = *(StatMu.InstanceStat.CacheHits) + uint64(1)
	StatMu.Mutex.Unlock()

}

// Atomically increments request delta in the instance stat
func incrCacheSetStatDelta() {
	StatMu.Mutex.Lock()

	// increment the CacheSets counter
	*(StatMu.InstanceStat.CacheSets) = *(StatMu.InstanceStat.CacheSets) + uint64(1)
	StatMu.Mutex.Unlock()

}
