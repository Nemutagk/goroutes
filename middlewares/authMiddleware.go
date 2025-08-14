package middlewares

import (
	"context"
	"net/http"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/golog"
	"github.com/Nemutagk/goroutes/definitions"
	"github.com/Nemutagk/goroutes/helper"
	"github.com/Nemutagk/goroutes/service"
)

type contextKey string

func AuthMiddleware(next http.HandlerFunc, route definitions.Route, dbListConn map[string]db.DbConnection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		golog.Log(context.Background(), "==================> AuthMiddleware called")
		if route.Auth == nil {
			golog.Log(context.Background(), "No auth middleware defined for this route, allowing access")

			next(w, r)
			return
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			golog.Error(context.Background(), "No token provided, denying access")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		res, err := service.AccountService("/auth/validation", "POST", map[string]string{
			"token":      token,
			"app":        route.Auth.App,
			"permission": route.Auth.Permission,
		})

		if err != nil {
			if httpErr, ok := err.(*service.HTTPError); ok {
				golog.Error(context.Background(), "Error from account service:", httpErr.Status)
				helper.PrettyPrint(httpErr)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(httpErr.Code)
				w.Write([]byte(httpErr.Body))
				return
			}

			golog.Error(context.Background(), "Error validating token:", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), contextKey("auth"), res)

		next(w, r.WithContext(ctx))
	}
}
