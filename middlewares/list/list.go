package list

import (
	"github.com/Nemutagk/goroutes/definitions"
	"github.com/Nemutagk/goroutes/middlewares"
)

var ListDefaultMiddlewares = []definitions.Middleware{
	middlewares.AccessMiddleware,
	middlewares.MethodMiddleware,
	middlewares.AuthMiddleware,
}
