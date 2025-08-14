package middlewares

import (
	"net/http"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/golog"
	"github.com/Nemutagk/goroutes/definitions"
)

func MethodMiddleware(next http.HandlerFunc, route definitions.Route, dbListConn map[string]db.DbConnection) http.HandlerFunc {
	return func(wr http.ResponseWriter, r *http.Request) {
		golog.Log(r.Context(), "==================> MethodMiddleware called")
		if r.Method != route.Method {
			golog.Log(r.Context(), "==================> MethodMiddleware END (method not allowed)")
			http.Error(wr, "", http.StatusNotFound)
			return
		}
		golog.Log(r.Context(), "==================> MethodMiddleware END")
		next(wr, r)
	}
}
