package middlewares

import (
	"net/http"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/golog"
	"github.com/Nemutagk/goroutes/definitions"
)

func CorsMiddleware(next http.HandlerFunc, route definitions.Route, dbListConn map[string]db.DbConnection) http.HandlerFunc {
	return func(wr http.ResponseWriter, r *http.Request) {
		golog.Log(r.Context(), "==================> CORS Middleware called")
		wr.Header().Set("Access-Control-Allow-Origin", "*")
		wr.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		wr.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Request-Timestamp, x-request-timestamp, Accept, Origin, User-Agent, Cache-Control")

		if r.Method == http.MethodOptions {
			golog.Log(r.Context(), "==================> CORS Middleware END (preflight)")
			wr.WriteHeader(http.StatusOK)
			wr.Write([]byte("OK"))
			return
		}

		golog.Log(r.Context(), "==================> CORS Middleware END")
		next(wr, r)
	}
}
