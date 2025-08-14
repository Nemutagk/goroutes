package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/golog"
	"github.com/Nemutagk/goroutes/definitions"
	"github.com/gofrs/uuid"
)

func InfoMiddleware(next http.HandlerFunc, route definitions.Route, dbListConn map[string]db.DbConnection) http.HandlerFunc {
	return func(wr http.ResponseWriter, r *http.Request) {
		golog.Log(r.Context(), "==================> InfoMiddleware called")

		clientRealIp := r.Header.Get("X-Forwarded-For")
		if clientRealIp == "" {
			clientRealIp = r.RemoteAddr
		}

		clientIp := clientRealIp
		if strings.Contains(clientIp, ",") {
			clientIp = strings.Split(clientIp, ",")[0]
		}
		if strings.Contains(clientIp, ":") {
			clientIp = strings.Split(clientIp, ":")[0]
		}

		golog.Log(r.Context(), "Client IP:"+clientIp)
		golog.Log(r.Context(), "Route path:"+r.URL.String())
		golog.Log(r.Context(), "Route method:"+route.Method)

		requestId := uuid.Must(uuid.NewV7()).String()
		if rid := r.Header.Get("X-RequestKb-ID"); rid != "" {
			requestId = rid
		}

		ctx := context.WithValue(r.Context(), "request_id", requestId)
		golog.Log(ctx, "Generated request ID:", requestId)
		golog.Log(ctx, "==================> InfoMiddleware called END")
		next(wr, r.WithContext(ctx))
	}
}
