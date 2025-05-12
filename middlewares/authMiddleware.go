package middlewares

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Nemutagk/goroutes/definitions"
	"github.com/Nemutagk/goroutes/helper"
	"github.com/Nemutagk/goroutes/service"
)

type contextKey string

func AuthMiddleware(next http.HandlerFunc, route definitions.Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if route.Auth == nil {
			fmt.Println("No auth middleware defined for this route, allowing access")

			next(w, r)
			return
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			fmt.Println("No token provided, denying access")
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
				fmt.Println("Error from account service:", httpErr.Status)
				helper.PrettyPrint(string(httpErr.Body))
				helper.PrettyPrint(string(httpErr.Error()))
			}

			fmt.Println("Error validating token:", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), contextKey("auth"), res)

		next(w, r.WithContext(ctx))
	}
}
