package notfound

import (
	"log"
	"net/http"
)

type ResponseRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	wroteBody   bool
}

func (r *ResponseRecorder) WriteHeader(code int) {
	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
	r.wroteBody = true
	return r.ResponseWriter.Write(b)
}

func CustomMuxHandler(mux *http.ServeMux, notFoundHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Crear un ResponseRecorder temporal para capturar la respuesta del mux
		rec := &ResponseRecorder{ResponseWriter: w, status: 200}
		mux.ServeHTTP(rec, r)
		log.Println("Request:", r.Method, r.URL.Path, "Status Code:", rec.status)
		if rec.status == 404 {
			notFoundHandler(w, r)
		}
	}
}
