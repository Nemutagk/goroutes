package middlewares

import (
	"context"
	"net/http"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/golog"
	"github.com/Nemutagk/goroutes/definitions"
)

func MethodMiddleware(next http.HandlerFunc, route definitions.Route, dbListConn map[string]db.DbConnection) http.HandlerFunc {
	// Check if the request method is allowed
	return func(wr http.ResponseWriter, r *http.Request) {
		golog.Log(context.Background(), "==================> MethodMiddleware called")
		if r.Method != route.Method {
			golog.Log(context.Background(), "==================> MethodMiddleware called END")
			http.Error(wr, "", http.StatusNotFound)
			return
		}

		golog.Log(context.Background(), "==================> MethodMiddleware called END")
		next(wr, r)
	}
}
