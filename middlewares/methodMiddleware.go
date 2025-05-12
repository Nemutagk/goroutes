package middlewares

import (
	"context"
	"net/http"

	"github.com/Nemutagk/goroutes/definitions"
)

func MethodMiddleware(next http.HandlerFunc, route definitions.Route, cxt context.Context) http.HandlerFunc {
	// Check if the request method is allowed
	return func(wr http.ResponseWriter, r *http.Request) {
		if r.Method != route.Method {
			http.Error(wr, "", http.StatusNotFound)
			return
		}
		next(wr, r)
	}
}
