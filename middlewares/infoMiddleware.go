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
		golog.Log(context.Background(), "==================> InfoMiddleware called")

		clientRealIp := r.Header.Get("X-Forwarded-For")
		if clientRealIp == "" {
			clientRealIp = r.RemoteAddr
		}

		var clientIp = clientRealIp

		if strings.Contains(clientIp, ",") {
			clientIp = strings.Split(clientIp, ",")[0]
		}

		if strings.Contains(clientIp, ":") {
			clientIp = strings.Split(clientIp, ":")[0]
		}

		golog.Log(context.Background(), "Client IP:"+clientIp)
		golog.Log(context.Background(), "Route path:"+r.URL.String())
		golog.Log(context.Background(), "Route method:"+route.Method)

		// Generate unique request id with uuid v7

		requestId := uuid.Must(uuid.NewV7()).String()
		requestIdHeader := r.Header.Get("X-RequestKb-ID")
		if requestIdHeader != "" {
			requestId = requestIdHeader
		}

		golog.Log(context.Background(), "Generated request ID:", requestId)
		ctx := context.WithValue(r.Context(), "request_id", requestId)
		r = r.WithContext(ctx)

		golog.Log(context.Background(), "==================> InfoMiddleware called END")
		next(wr, r)
	}
}
