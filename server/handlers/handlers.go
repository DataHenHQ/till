package handlers

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/DataHenHQ/till/internal/tillclient"
	"github.com/DataHenHQ/tillup/logger"
	"github.com/DataHenHQ/tillup/sessions"
	"github.com/unrolled/render"
	"github.com/volatiletech/null/v8"
)

var (
	// rend is html/json/xml renderer
	Rend *render.Render

	CurrentInstance *tillclient.Instance

	StatMu *tillclient.InstanceStatMutex

	InstanceName string

	LoggerConfig logger.Config
)

func SetEmbeddedTemplates(e *embed.FS) {
	GIDShaRe := regexp.MustCompile(`-([a-zA-Z0-9]+)$`)

	templateFunc := template.FuncMap{
		"LoggerConfig": func() logger.Config {
			return LoggerConfig
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

	Rend = render.New(render.Options{
		Extensions: []string{".tmpl", ".html"},
		FileSystem: &render.EmbedFileSystem{
			FS: *e,
		},
		Funcs: []template.FuncMap{templateFunc},
	})

}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	Rend.HTML(w, http.StatusOK, "index", map[string]interface{}{
		"Instance":            InstanceName,
		"Requests":            CurrentInstance.GetRequests() + int64(*StatMu.Requests),
		"InterceptedRequests": CurrentInstance.GetInterceptedRequests() + int64(*StatMu.InterceptedRequests),
		"FailedRequests":      CurrentInstance.GetFailedRequests() + int64(*StatMu.FailedRequests),
		"CacheHits":           CurrentInstance.GetCacheHits() + int64(*StatMu.CacheHits),
		"CacheSets":           CurrentInstance.GetCacheSets() + int64(*StatMu.CacheSets),
	})
}
