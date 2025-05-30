package definitions

import "net/http"

type RouteGroup struct {
	Prefix      string
	Middlewares *[]Middleware
	Routes      []interface{}
}

type Route struct {
	Path               string
	Method             string
	Action             http.HandlerFunc
	Middlewares        *[]Middleware
	MiddlewareParams   *map[string]interface{}
	ExcludeMiddlewares *[]Middleware
	Auth               *RouteAuth
	Group              map[string]Route
}

type RouteAuth struct {
	App        string
	Permission string
}
