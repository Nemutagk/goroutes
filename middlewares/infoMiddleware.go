package middlewares

import (
	"log"
	"net/http"
	"strings"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/goroutes/definitions"
)

func InfoMiddleware(next http.HandlerFunc, route definitions.Route, dbListConn map[string]db.DbConnection) http.HandlerFunc {
	return func(wr http.ResponseWriter, r *http.Request) {
		log.Println("==================> InfoMiddleware called")

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

		log.Printf("Client IP: %s\n", clientIp)
		log.Println("Route path:" + r.URL.String())
		log.Println("Route method:" + route.Method)
		log.Println("==================> InfoMiddleware called ending")

		next(wr, r)
	}
}
