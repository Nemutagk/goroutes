package middlewares

import (
	"context"
	"net/http"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/golog"
	"github.com/Nemutagk/goroutes/definitions"
)

func CorsMiddleware(next http.HandlerFunc, route definitions.Route, dbListConn map[string]db.DbConnection) http.HandlerFunc {
	return http.HandlerFunc(func(wr http.ResponseWriter, r *http.Request) {
		golog.Log(context.Background(), "==================> CORS Middleware called")
		golog.Log(context.Background(), "CORS Middleware called")
		wr.Header().Set("Access-Control-Allow-Origin", "*")
		wr.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		wr.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Request-Timestamp, x-request-timestamp, Accept, Origin, User-Agent, Cache-Control")

		if r.Method == http.MethodOptions {
			wr.WriteHeader(http.StatusOK)
			wr.Write([]byte("OK"))
			return
		}

		next(wr, r)
	})
}
