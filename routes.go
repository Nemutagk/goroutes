package goroutes

import (
	"fmt"
	"net/http"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/goroutes/definitions"
)

func LoadRoutes(list_routes []definitions.RouteGroup, server *http.ServeMux, defaultMiddlewares []definitions.Middleware, notFoundHandler http.HandlerFunc, dbConnectionsList map[string]db.DbConnection) *http.ServeMux {
	globalRouteList := map[string]definitions.Route{}

	for _, groupRoute := range list_routes {
		routes := checkRouteGroup(groupRoute, "")

		for tmp_path, tmp_route := range routes {
			globalRouteList[tmp_path] = tmp_route
		}
	}

	for path, route := range globalRouteList {
		if route.Group == nil {
			route = validateMiddleware(route, defaultMiddlewares)
		} else {
			for key_subroute, sub_route := range route.Group {
				sub_route = validateMiddleware(sub_route, defaultMiddlewares)
				route.Group[key_subroute] = sub_route
			}
		}

		// fmt.Println("Route: ", path, "Method: ", route.Method)
		server.HandleFunc(path, applyMiddleware(route, dbConnectionsList))
	}

	server.HandleFunc("/notfound", func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/notfound" {
			fmt.Println("The route " + req.URL.Path + " and method " + req.Method + " does not mapped in the routes")
			notFoundHandler(res, req)
			return
		}
	})

	return server
}

func checkRouteGroup(routeGroup definitions.RouteGroup, parentPath string) map[string]definitions.Route {
	path := preparePath(routeGroup.Prefix, parentPath)

	route_list := make(map[string]definitions.Route)

	for _, route := range routeGroup.Routes {
		if subroute, ok := route.(definitions.RouteGroup); ok {

			sub_routes := checkRouteGroup(subroute, path)
			for sub_path, sub_route := range sub_routes {
				route_list = routeExists(route_list, sub_path, sub_route)
			}

			continue
		}

		route_define, ok := route.(definitions.Route)
		if !ok {
			fmt.Println("Invalid route definition:", route)
			continue
		}

		sub_path := preparePath(route_define.Path, path)

		route_list = routeExists(route_list, sub_path, route_define)
	}

	return route_list
}

func preparePath(prefix string, parentPath string) string {
	path := parentPath + prefix

	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	if path == "" {
		path = "/"
	}

	return path
}

func routeExists(routes map[string]definitions.Route, path string, route definitions.Route) map[string]definitions.Route {
	if _, exists := routes[path]; !exists {
		routes[path] = route
	} else {
		tmp_route := routes[path]

		if tmp_route.Group == nil {
			tmp_route.Group = make(map[string]definitions.Route)
			tmp_route.Group[tmp_route.Method] = definitions.Route{
				Path:               tmp_route.Path,
				Method:             tmp_route.Method,
				Action:             tmp_route.Action,
				Middlewares:        tmp_route.Middlewares,
				ExcludeMiddlewares: tmp_route.ExcludeMiddlewares,
				Auth:               tmp_route.Auth,
			}
			tmp_route.Group[route.Method] = route
		} else {
			if _, exists := tmp_route.Group[route.Method]; !exists {
				tmp_route.Group[route.Method] = route
			} else {
				fmt.Println("Route already exists:", route.Method, route.Path)
			}
		}

		routes[path] = tmp_route
	}

	return routes
}

func containsMiddleware(middleware []definitions.Middleware, mw definitions.Middleware) bool {
	for _, m := range middleware {
		if fmt.Sprintf("%p", m) == fmt.Sprintf("%p", mw) {
			return true
		}
	}

	return false
}

func validateMiddleware(route definitions.Route, defaultMiddleware []definitions.Middleware) definitions.Route {
	if len(route.Middlewares) == 0 {
		if route.ExcludeMiddlewares == nil || len(*route.ExcludeMiddlewares) == 0 {
			route.Middlewares = defaultMiddleware
		} else {
			for _, mw := range defaultMiddleware {
				if !containsMiddleware(*route.ExcludeMiddlewares, mw) {
					route.Middlewares = append(route.Middlewares, mw)
				}
			}
		}

		return route
	}

	for _, mw := range defaultMiddleware {
		if !containsMiddleware(route.Middlewares, mw) && !containsMiddleware(*route.ExcludeMiddlewares, mw) {
			route.Middlewares = append(route.Middlewares, mw)
		}
	}

	return route
}

func applyMiddleware(route definitions.Route, dbListConn map[string]db.DbConnection) http.HandlerFunc {
	if route.Group == nil {
		for _, middleware := range route.Middlewares {
			route.Action = middleware(route.Action, route, dbListConn)
		}

		return route.Action
	} else {
		return func(res http.ResponseWriter, req *http.Request) {
			if sub_route, exists := route.Group[req.Method]; exists {
				for _, middleware := range sub_route.Middlewares {
					sub_route.Action = middleware(sub_route.Action, sub_route, dbListConn)
				}

				sub_route.Action(res, req)
				return
			}

			if req.Method == http.MethodOptions {
				fmt.Println("CORS preflight request")
				res.Header().Set("Access-Control-Allow-Origin", "*")
				res.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				res.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

				fmt.Println("CORS preflight request")
				res.WriteHeader(http.StatusOK)
				return
			}

			fmt.Println("the methods does not exists in the group!")
			fmt.Println("The route " + route.Path + " and method " + req.Method + " does not mapped in the routes")

			res.WriteHeader(http.StatusNotFound)
			res.Header().Set("Content-Type", "application/json")
			res.Write([]byte(""))
		}
	}
}
