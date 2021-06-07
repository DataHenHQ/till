package handlers

import (
	"embed"
	"html/template"
	"net/http"
	"net/url"

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

	templateFunc := template.FuncMap{
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
