package handlers

import (
	"fmt"
	"net/http"

	"github.com/DataHenHQ/tillup/logger"
	"github.com/gorilla/mux"
)

func RequestIndexHandler(w http.ResponseWriter, r *http.Request) {

	var startAfter, endBefore string
	if sa, ok := r.URL.Query()["start_after"]; ok && len(sa) == 1 {
		startAfter = sa[0]
	}
	if eb, ok := r.URL.Query()["end_before"]; ok && len(eb) == 1 {
		endBefore = eb[0]
	}

	is, p, err := logger.GetItems(100, startAfter, endBefore)
	if err != nil {
		fmt.Println("error on requests", err)
	}
	Rend.HTML(w, http.StatusOK, "requests/index", map[string]interface{}{
		"Items":      is,
		"Pagination": p,
	})
}

func RequestShowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rid := vars["rid"]

	i, err := logger.GetItem(rid)
	if err != nil {
		fmt.Println("error on requests", err)
	}

	Rend.HTML(w, http.StatusOK, "requests/show", map[string]interface{}{
		"CacheHit":  i.CacheHit,
		"RID":       i.RID,
		"GID":       i.GID,
		"CreatedAt": i.CreatedAt,
		"Request":   i.Request,
		"Response":  i.Response,
	})
}

func RequestContentShowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rid := vars["rid"]

	i, err := logger.GetItem(rid)
	if err != nil {
		fmt.Println("error on requests", err)
	}

	// hijack the response writer, to get the raw Connection
	rawConn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		http.Error(w, "error writing content", 500)
		return
	}
	defer rawConn.Close()

	// build the HTTP response
	resp := i.Response.BuildHTTPResponse()

	// does a raw write of the response into the connection
	resp.Write(rawConn)

}
