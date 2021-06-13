package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/DataHenHQ/tillup/logger"
	"github.com/gorilla/mux"
)

func RequestIndexHandler(w http.ResponseWriter, r *http.Request) {

	var (
		f          = logger.Filter{}
		is         []logger.Item
		p          logger.Pagination
		err        error
		perPage    = 100
		startAfter string
		endBefore  string
	)

	if q, ok := r.URL.Query()["from_content_length"]; ok && len(q) == 1 {
		v, _ := strconv.ParseInt(q[0], 10, 64)
		if err == nil {
			f.FromResponseContentLength = &v
		}
	}
	if q, ok := r.URL.Query()["to_content_length"]; ok && len(q) == 1 {
		v, _ := strconv.ParseInt(q[0], 10, 64)
		if err == nil {
			f.ToResponseSize = &v
		}
	}
	if q, ok := r.URL.Query()["from_time"]; ok && len(q) == 1 {
		v, err := time.Parse(time.RFC3339, q[0])
		if err == nil {
			f.FromTime = v
		}
	}
	if q, ok := r.URL.Query()["to_time"]; ok && len(q) == 1 {
		v, err := time.Parse(time.RFC3339, q[0])
		if err == nil {
			f.ToTime = v
		}
	}
	if q, ok := r.URL.Query()["code"]; ok && len(q) == 1 {
		f.Code = q[0]
	}
	if q, ok := r.URL.Query()["gid"]; ok && len(q) == 1 {
		f.GID = q[0]
	}
	if q, ok := r.URL.Query()["cache"]; ok && len(q) == 1 {
		f.Cache = q[0]
	}
	if q, ok := r.URL.Query()["method"]; ok && len(q) == 1 {
		f.Method = q[0]
	}
	if q, ok := r.URL.Query()["start_after"]; ok && len(q) == 1 {
		startAfter = q[0]
	}
	if q, ok := r.URL.Query()["end_before"]; ok && len(q) == 1 {
		endBefore = q[0]
	}

	is, p, err = logger.GetItems(f, perPage, startAfter, endBefore)
	if err != nil {
		fmt.Println("error on requests", err)
	}

	Rend.HTML(w, http.StatusOK, "requests/index", map[string]interface{}{
		"Items":      is,
		"Pagination": p,
		"CurrentURL": r.URL.RequestURI(),
		"Filter":     f,
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
		"Item": i,
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
	resp := i.BuildHTTPResponse()

	// does a raw write of the response into the connection
	resp.Write(rawConn)

}
