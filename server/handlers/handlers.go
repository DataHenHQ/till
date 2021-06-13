package handlers

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"

	"github.com/DataHenHQ/till/internal/tillclient"
	"github.com/unrolled/render"
)

var (
	// rend is html/json/xml renderer
	Rend *render.Render

	CurrentInstance *tillclient.Instance

	StatMu *tillclient.InstanceStatMutex

	InstanceName string
)

func SetEmbeddedTemplates(e *embed.FS) {
	GIDShaRe := regexp.MustCompile(`-([a-zA-Z0-9]+)$`)

	templateFunc := template.FuncMap{
		"unescape": func(s string) template.HTML {
			return template.HTML(s)
		},
		"shortGID": func(gid string) string {
			sha := GIDShaRe.FindStringSubmatch(gid)
			// sha := ss[1]

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
				case int:
					si, _ := v.(int)
					s = fmt.Sprintf("%d", si)
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
