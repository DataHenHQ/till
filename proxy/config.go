package proxy

import (
	"net/http"
	"strconv"

	"github.com/DataHenHQ/tillup/sessions/sticky"
)

// PageConfig is where the page configuration is set
type PageConfig struct {
	ForceUA  bool   // if true, overrides the User-Agent header
	UaType   string // default to "desktop"
	UseProxy bool

	// StickySession features
	SessionID     string
	StickyCookies bool
	StickyUA      bool

	// Interceptors feature
	IgnoreInterceptors    []string
	IgnoreAllInterceptors bool
}

// UATypeHeader is the custom header that the scraper calls till to set the user agent type
const UATypeHeader = "X-DH-UA-Type"

func generatePageConfig(req *http.Request) (pconf *PageConfig) {
	useProxy := false
	if ProxyCount > 0 {
		useProxy = true
	}

	pconf = &PageConfig{
		ForceUA:  ForceUA,
		UaType:   UAType,
		UseProxy: useProxy,

		// StickySessions feature
		StickyCookies: true,
		StickyUA:      true,

		// Interceptors feature
		IgnoreInterceptors:    []string{},
		IgnoreAllInterceptors: false,
	}

	if uatype := req.Header.Get(UATypeHeader); uatype != "" {
		pconf.UaType = uatype
		req.Header.Del(UATypeHeader)
	}

	// Get the session ID header
	if sessid := req.Header.Get(sticky.SessionIDHeader); sessid != "" {
		pconf.SessionID = sessid
		req.Header.Del(sticky.SessionIDHeader)
	}

	// Get the Sticky UA header
	if val := req.Header.Get(sticky.StickyUAHeader); val != "" {
		pconf.StickyUA, _ = strconv.ParseBool(val)
		req.Header.Del(sticky.StickyUAHeader)
	}

	// Get the Sticky Cookies header
	if val := req.Header.Get(sticky.StickyCookiesHeader); val != "" {
		pconf.StickyCookies, _ = strconv.ParseBool(val)
		req.Header.Del(sticky.StickyCookiesHeader)
	}
	defer req.Header.Del(sticky.StickyCookiesHeader)

	return pconf
}
