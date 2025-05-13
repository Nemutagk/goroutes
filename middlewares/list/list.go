package list

import (
	"github.com/Nemutagk/goroutes/definitions"
	"github.com/Nemutagk/goroutes/middlewares"
)

var ListDefaultMiddlewares = []definitions.Middleware{
	middlewares.AuthMiddleware,
	middlewares.MethodMiddleware,
	middlewares.AccessMiddleware,
	middlewares.CorsMiddleware,
}
