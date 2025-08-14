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
		golog.Log(r.Context(), "==================> AuthMiddleware called")
		if route.Auth == nil {
			golog.Log(r.Context(), "No auth middleware defined for this route, allowing access")
			golog.Log(r.Context(), "==================> AuthMiddleware END")

			next(w, r)
			return
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			golog.Error(r.Context(), "No token provided, denying access")
			golog.Log(r.Context(), "==================> AuthMiddleware END")
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
				golog.Error(r.Context(), "Error from account service:", httpErr.Status)
				helper.PrettyPrint(httpErr)
				golog.Log(r.Context(), "==================> AuthMiddleware END")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(httpErr.Code)
				w.Write([]byte(httpErr.Body))
				return
			}

			golog.Error(r.Context(), "Error validating token:", err)
			golog.Log(r.Context(), "==================> AuthMiddleware END")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), contextKey("auth"), res)
		golog.Log(ctx, "==================> AuthMiddleware END")

		next(w, r.WithContext(ctx))
	}
}
