package middlewares

import (
	"fmt"
	"net/http"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/goroutes/definitions"
)

func MethodMiddleware(next http.HandlerFunc, route definitions.Route, dbListConn map[string]db.DbConnection) http.HandlerFunc {
	// Check if the request method is allowed
	return func(wr http.ResponseWriter, r *http.Request) {
		fmt.Println("MethodMiddleware called")
		if r.Method != route.Method {
			http.Error(wr, "", http.StatusNotFound)
			return
		}
		next(wr, r)
	}
}
