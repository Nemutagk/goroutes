package goroutes

import (
	"fmt"
	"net/http"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/goroutes/definitions"
)

func LoadRoutes(list_routes []definitions.RouteGroup, server *http.ServeMux, notFoundHandler http.HandlerFunc, dbConnectionsList map[string]db.DbConnection) *http.ServeMux {
	globalRouteList := map[string]definitions.Route{}

	for _, groupRoute := range list_routes {
		routes := checkRouteGroup(groupRoute, "", nil)

		for tmp_path, tmp_route := range routes {
			globalRouteList[tmp_path] = tmp_route
		}
	}

	for path, route := range globalRouteList {
		server.HandleFunc(path, applyMiddleware(route, dbConnectionsList))
	}

	server.HandleFunc("/404", notFoundHandler)

	return server
}

func checkRouteGroup(routeGroup definitions.RouteGroup, parentPath string, parentMiddleware []definitions.Middleware) map[string]definitions.Route {
	path := preparePath(routeGroup.Prefix, parentPath)

	if routeGroup.Middlewares != nil && len(*routeGroup.Middlewares) > 0 {
		fmt.Println("route group issues middleware!")
		parentMiddleware = append(parentMiddleware, *routeGroup.Middlewares...)
	}

	route_list := make(map[string]definitions.Route)

	for _, route := range routeGroup.Routes {
		if subroute, ok := route.(definitions.RouteGroup); ok {

			sub_routes := checkRouteGroup(subroute, path, parentMiddleware)
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

		if parentMiddleware != nil && len(parentMiddleware) > 0 {
			if route_define.Middlewares == nil {
				route_define.Middlewares = &parentMiddleware
			} else {
				mws := append(*route_define.Middlewares, parentMiddleware...)
				route_define.Middlewares = &mws
			}
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

func applyMiddleware(route definitions.Route, dbListConn map[string]db.DbConnection) http.HandlerFunc {
	if route.Group == nil {
		if route.Middlewares != nil && len(*route.Middlewares) > 0 {
			for _, middleware := range *route.Middlewares {
				route.Action = middleware(route.Action, route, dbListConn)
			}
		}

		return route.Action
	} else {
		return func(res http.ResponseWriter, req *http.Request) {
			if sub_route, exists := route.Group[req.Method]; exists {
				for _, middleware := range *sub_route.Middlewares {
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
