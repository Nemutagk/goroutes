package wr

import (
	"bytes"
	"net/http"
)

type responseWriterRecorder struct {
	rw          http.ResponseWriter
	status      int
	wroteHeader bool
	body        bytes.Buffer
}

func NewResponseRecorder(rw http.ResponseWriter) *responseWriterRecorder {
	return &responseWriterRecorder{rw: rw, status: http.StatusOK}
}

func (r *responseWriterRecorder) Header() http.Header {
	return r.rw.Header()
}

func (r *responseWriterRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.status = code
	r.wroteHeader = true
	r.rw.WriteHeader(code)
}

func (r *responseWriterRecorder) Write(b []byte) (int, error) {
	// forward to underlying writer and record bytes
	n, err := r.rw.Write(b)
	if n > 0 {
		r.body.Write(b[:n])
	}
	// if Write was used without WriteHeader, ensure status is set
	if !r.wroteHeader {
		r.wroteHeader = true
		r.status = http.StatusOK
	}
	return n, err
}

// optional: forward Flush if underlying writer supports it
func (r *responseWriterRecorder) Flush() {
	if f, ok := r.rw.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *responseWriterRecorder) GetStatus() int {
	return r.status
}
