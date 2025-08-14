package notfound

import (
	"bytes"
	"net/http"
)

type ResponseRecorder struct {
	status      int
	wroteHeader bool
	body        bytes.Buffer
	header      http.Header
}

func NewResponseRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		header: make(http.Header),
	}
}

func (r *ResponseRecorder) Header() http.Header {
	return r.header
}

func (r *ResponseRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.status = code
	r.wroteHeader = true
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.body.Write(b)
}

func (r *ResponseRecorder) Status() int {
	return r.status
}

func CustomMuxHandler(mux *http.ServeMux, notFoundHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h, pattern := mux.Handler(r)

		// Fallback: patrón "/" para paths distintos de "/" (cuando "/" actúa como catch-all)
		if r.URL.Path != "/" && pattern == "/" {
			notFoundHandler(w, r)
			return
		}

		rr := NewResponseRecorder()
		h.ServeHTTP(rr, r)
		if rr.status == 0 {
			rr.status = http.StatusOK
		}

		// Solo si NO existe ruta realmente (pattern == "") usamos notFoundHandler
		if rr.status == http.StatusNotFound && pattern == "" {
			notFoundHandler(w, r)
			return
		}

		for k, vals := range rr.header {
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(rr.status)
		_, _ = w.Write(rr.body.Bytes())
	}
}
