package goroutes

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/goroutes/definitions"
	"github.com/Nemutagk/goroutes/definitions/notfound"
	"github.com/Nemutagk/goroutes/helper"
	"github.com/Nemutagk/goroutes/middlewares"
)

func LoadRoutes(list_routes []definitions.RouteGroup, server *http.ServeMux, dbConnectionsList map[string]db.DbConnection) *http.ServeMux {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	globalRouteList := map[string]definitions.Route{}

	for _, groupRoute := range list_routes {
		routes := checkRouteGroup(groupRoute, "", nil)

		for tmp_path, tmp_route := range routes {
			globalRouteList[tmp_path] = tmp_route
		}
	}

	totalRouteList := make([]string, 0)
	for path, route := range globalRouteList {
		totalRouteList = append(totalRouteList, route.Method+": "+path)
		server.HandleFunc(path, applyMiddleware(route, dbConnectionsList))
	}

	fmt.Println("Routes loaded successfully")
	helper.PrettyPrint(totalRouteList)

	return server
}

func AddNotFoundHandler(server *http.ServeMux, notFoundHandler http.HandlerFunc) http.HandlerFunc {
	return notfound.CustomMuxHandler(server, notFoundHandler)
}

func checkRouteGroup(routeGroup definitions.RouteGroup, parentPath string, parentMiddleware []definitions.Middleware) map[string]definitions.Route {
	path := preparePath(routeGroup.Prefix, parentPath)

	if routeGroup.Middlewares != nil && len(*routeGroup.Middlewares) > 0 {
		// fmt.Println("route group issues middleware!")
		parentMiddleware = append(parentMiddleware, *routeGroup.Middlewares...)
	}

	routeList := make(map[string]definitions.Route)

	for _, route := range routeGroup.Routes {
		if subroute, ok := route.(definitions.RouteGroup); ok {

			subRoutes := checkRouteGroup(subroute, path, parentMiddleware)
			for sub_path, sub_route := range subRoutes {
				routeList = routeExists(routeList, sub_path, sub_route)
			}

			continue
		}

		routeDefine, ok := route.(definitions.Route)
		if !ok {
			fmt.Println("Invalid route definition:", route)
			continue
		}

		if len(parentMiddleware) > 0 {
			if routeDefine.Middlewares == nil {
				routeDefine.Middlewares = &parentMiddleware
			} else {
				mws := append(*routeDefine.Middlewares, parentMiddleware...)
				routeDefine.Middlewares = &mws
			}
		}

		subPath := preparePath(routeDefine.Path, path)

		routeList = routeExists(routeList, subPath, routeDefine)
	}

	return routeList
}

func preparePath(prefix string, parentPath string) string {
	if parentPath == "/" {
		if prefix != "" && prefix[0:1] == "/" {
			prefix = prefix[1:]
		}
	}

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
		tmpRoute := routes[path]

		if tmpRoute.Group == nil {
			tmpRoute.Group = make(map[string]definitions.Route)
			tmpRoute.Group[tmpRoute.Method] = definitions.Route{
				Path:               tmpRoute.Path,
				Method:             tmpRoute.Method,
				Action:             tmpRoute.Action,
				Middlewares:        tmpRoute.Middlewares,
				ExcludeMiddlewares: tmpRoute.ExcludeMiddlewares,
				Auth:               tmpRoute.Auth,
			}
			tmpRoute.Group[route.Method] = route
		} else {
			if _, exists := tmpRoute.Group[route.Method]; !exists {
				tmpRoute.Group[route.Method] = route
			} else {
				fmt.Println("Route already exists:", route.Method, route.Path)
			}
		}

		routes[path] = tmpRoute
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
		route_list_middleware := *route.Middlewares
		if route_list_middleware == nil {
			route_list_middleware = make([]definitions.Middleware, 0)
			route_list_middleware = append(route_list_middleware, middlewares.InfoMiddleware)
			route.Middlewares = &route_list_middleware
		} else {
			route_list_middleware = append(route_list_middleware, middlewares.InfoMiddleware)
			route.Middlewares = &route_list_middleware
		}

		if route.Middlewares != nil && len(*route.Middlewares) > 0 {
			for _, middleware := range *route.Middlewares {
				route.Action = middleware(route.Action, route, dbListConn)
			}
		}

		return route.Action
	} else {
		return func(res http.ResponseWriter, req *http.Request) {
			if sub_route, exists := route.Group[req.Method]; exists {
				route_list_middleware := *sub_route.Middlewares
				if route_list_middleware == nil {
					route_list_middleware = make([]definitions.Middleware, 0)
					route_list_middleware = append(route_list_middleware, middlewares.InfoMiddleware)
					sub_route.Middlewares = &route_list_middleware
				} else {
					route_list_middleware = append(route_list_middleware, middlewares.InfoMiddleware)
					sub_route.Middlewares = &route_list_middleware
				}

				if sub_route.Middlewares != nil && len(*sub_route.Middlewares) > 0 {
					for _, middleware := range *sub_route.Middlewares {
						sub_route.Action = middleware(sub_route.Action, sub_route, dbListConn)
					}
				}

				sub_route.Action(res, req)
				return
			}

			if req.Method == http.MethodOptions {
				fmt.Println("CORS preflight request")
				res.Header().Set("Access-Control-Allow-Origin", "*")
				res.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				res.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Request-Timestamp, x-request-timestamp, Accept, Origin, User-Agent, Cache-Control")

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
