package handlers

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/DataHenHQ/till/internal/tillclient"
	"github.com/DataHenHQ/tillup/logger"
	"github.com/DataHenHQ/tillup/sessions"
	"github.com/foolin/goview"
	"github.com/volatiletech/null/v8"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	Renderer *goview.ViewEngine

	CurrentInstance *tillclient.Instance

	StatMu *tillclient.InstanceStatMutex

	InstanceName string

	LoggerConfig logger.Config

	lp = message.NewPrinter(language.English)
)

func SetEmbeddedTemplates(e *embed.FS) {
	GIDShaRe := regexp.MustCompile(`-([a-zA-Z0-9]+)$`)

	templateFunc := template.FuncMap{
		"LoggerConfig": func() logger.Config {
			return LoggerConfig
		},
		"InstanceName": func() string {
			return InstanceName
		},
		"jsonToHeader": func(nb null.Bytes) (h http.Header) {

			if nb.IsZero() {
				return nil
			}

			if err := json.Unmarshal([]byte(nb.Bytes), &h); err != nil {
				return nil
			}

			return h
		},
		"jsonToPageConfig": func(nb null.Bytes) (pconf sessions.PageConfig) {

			if nb.IsZero() {
				return pconf
			}

			if err := json.Unmarshal([]byte(nb.Bytes), &pconf); err != nil {
				return pconf
			}

			return pconf
		},
		"jsonToSession": func(nb null.Bytes) (sess sessions.Session) {

			if nb.IsZero() {
				return sess
			}

			if err := json.Unmarshal([]byte(nb.Bytes), &sess); err != nil {
				return sess
			}

			return sess
		},
		"jsonToSlice": func(nb null.Bytes) (ss []string) {

			if nb.IsZero() {
				return nil
			}

			if err := json.Unmarshal([]byte(nb.Bytes), &ss); err != nil {
				return nil
			}

			return ss
		},
		"nullGt": func(i null.Int64, exp int) bool {
			if i.IsZero() {
				return false
			}
			return i.Int64 > int64(exp)
		},
		"intToTime": func(i int64) time.Time {
			return time.Unix(0, i)
		},
		"intToBytes": func(b int64) string {
			const unit = 1000
			if b < unit {
				return fmt.Sprintf("%d B", b)
			}
			div, exp := int64(unit), 0
			for n := b / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return fmt.Sprintf("%.2f %cB",
				float64(b)/float64(div), "kMGTPE"[exp])
		},
		"ifThenElse": func(navinf interface{}, currentNav string, truecond string, falsecond string) string {
			nav, _ := navinf.(string)
			if nav == currentNav {
				return truecond
			}
			return falsecond
		},
		"localizeInt": func(i int64) string {
			return lp.Sprintf("%d\n", i)
		},
		"boolval": func(b *bool) bool {
			return *b
		},
		"unescape": func(s string) template.HTML {
			return template.HTML(s)
		},
		"shortGID": func(gid string) string {
			sha := GIDShaRe.FindStringSubmatch(gid)
			if len(sha) < 1 {
				return ""
			}

			return sha[1][0:5]
		},
		"isReqSuccess": func(code int64) (success bool) {
			switch {
			case code >= 200 && code <= 299:
				return true
			case code >= 300 && code <= 399:
				return true
			case code == 404:
				return true
			default:
				return false
			}
		},
		// converts url into relative path
		"relativepath": func(urlstr string) string {
			u, err := url.Parse(urlstr)
			if err != nil {
				return ""
			}
			u.Scheme = ""
			u.Host = ""
			u.User = nil
			return u.String()
		},
		// get the basepath of a url, basically the last item on the url
		"basepath": func(urlstr string) string {
			u, err := url.Parse(urlstr)
			if err != nil {
				return ""
			}

			return path.Base(u.Path)
		},
		// converts url into relative path
		"basepathPlusQ": func(urlstr string) string {
			u, err := url.Parse(urlstr)
			if err != nil {
				return ""
			}
			u.Scheme = ""
			u.Host = ""
			u.User = nil
			rp := u.String()
			ss := strings.Split(rp, "/")
			out := ss[len(ss)-1]
			if len(out) < 1 {
				out = "/"
			}
			return out
		},
		"basepathPlusQOrHost": func(urlstr string) string {
			u, err := url.Parse(urlstr)
			if err != nil {
				return ""
			}
			u.Scheme = ""
			host := u.Host
			u.Host = ""
			u.User = nil
			rp := u.String()
			ss := strings.Split(rp, "/")
			out := ss[len(ss)-1]
			if len(out) < 1 {
				out = host
			}
			return out
		},
		// get hostname and port
		"hostname": func(urlstr string) string {
			u, err := url.Parse(urlstr)
			if err != nil {
				return ""
			}

			return u.Hostname()
		},
		"appendQueryString": func(currurl string, keyvals ...interface{}) string {
			u, err := url.Parse(currurl)
			if err != nil {
				return ""
			}
			u.Scheme = ""
			u.Host = ""
			u.User = nil

			// TODO: Need to make this into key pairs, and set or delete based if the values is blank or not
			// get the queries
			qs := u.Query()

			kvs := map[string]string{}

			// create a map from keyvals
			cur := ""
			for i, v := range keyvals {
				// cast the types into the correct one
				var s string
				switch v.(type) {
				case string:
					s, _ = v.(string)
				case int, int64:
					s = fmt.Sprintf("%d", v)
				}

				// set it as the key or value based on mod 2
				if i%2 == 0 {
					cur = s
					kvs[cur] = ""
				} else {
					kvs[cur] = s
				}
			}

			// delete the key if the value is empty, otherwise set it
			for k, v := range kvs {
				if v == "" {
					qs.Del(k)
				} else {
					qs.Set(k, v)
				}
			}

			// Assign it back to the url
			u.RawQuery = qs.Encode()

			return u.String()
		},
	}

	gvConfig := goview.Config{
		Root:         "templates",
		Extension:    ".html",
		Master:       "layouts/master",
		DisableCache: false,
		Funcs:        templateFunc,
	}

	Renderer = goview.New(gvConfig)

	// set the filehandler for goview to use embedded FS
	Renderer.SetFileHandler(func(config goview.Config, tmpl string) (string, error) {
		path := filepath.Join(config.Root, tmpl)
		bytes, err := e.ReadFile(path + config.Extension)
		return string(bytes), err
	})

}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	Renderer.Render(w, http.StatusOK, "index", map[string]interface{}{
		"title":               "Home",
		"tab":                 "home",
		"Requests":            CurrentInstance.GetRequests() + int64(*StatMu.Requests),
		"FailedRequests":      CurrentInstance.GetFailedRequests() + int64(*StatMu.FailedRequests),
		"SuccessfulRequests":  CurrentInstance.GetSuccessfulRequests() + int64(*StatMu.SuccessfulRequests),
		"InterceptedRequests": CurrentInstance.GetInterceptedRequests() + int64(*StatMu.InterceptedRequests),
		"CacheHits":           CurrentInstance.GetCacheHits() + int64(*StatMu.CacheHits),
		"CacheSets":           CurrentInstance.GetCacheSets() + int64(*StatMu.CacheSets),
	})

}
