package middlewares

import (
	"net/http"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/goenvars"
	"github.com/Nemutagk/golog"
	"github.com/Nemutagk/goroutes/definitions"
)

func CorsMiddleware(next http.HandlerFunc, route definitions.Route, dbListConn map[string]db.DbConnection) http.HandlerFunc {
	return func(wr http.ResponseWriter, r *http.Request) {
		golog.Log(r.Context(), "==================> CORS Middleware called")
		wr.Header().Set("Access-Control-Allow-Origin", goenvars.GetEnv("CORS_ALLOW_ORIGIN", "*"))
		wr.Header().Set("Access-Control-Allow-Methods", goenvars.GetEnv("CORS_ALLOW_METHODS", "POST, GET, OPTIONS, PUT, DELETE"))
		wr.Header().Set("Access-Control-Allow-Headers", goenvars.GetEnv("CORS_ALLOW_HEADERS", "Content-Type, Authorization, X-Requested-With, X-Request-Timestamp, x-request-timestamp, Accept, Origin, User-Agent, Cache-Control"))
		wr.Header().Set("Access-Control-Expose-Headers", goenvars.GetEnv("CORS_EXPOSE_HEADERS", "Content-Length,Content-Type"))
		wr.Header().Set("Access-Control-Max-Age", goenvars.GetEnv("CORS_MAX_AGE", "86400"))
		if goenvars.GetEnvBool("CORS_ALLOW_CREDENTIALS", false) {
			wr.Header().Set("Access-Control-Allow-Credentials", "true")
		}

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
