package middlewares

import (
	"net/http"

	"github.com/Nemutagk/goroutes/definitions"
)

func MethodMiddleware(next http.HandlerFunc, route definitions.Route) http.HandlerFunc {
	// Check if the request method is allowed
	return func(wr http.ResponseWriter, r *http.Request) {
		if r.Method != route.Method {
			http.Error(wr, "", http.StatusNotFound)
			return
		}
		next(wr, r)
	}
}
