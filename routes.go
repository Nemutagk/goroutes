package goroutes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"slices"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/goenvars"
	"github.com/Nemutagk/goerrors"
	"github.com/Nemutagk/golog"
	"github.com/Nemutagk/goroutes/definitions"
	"github.com/Nemutagk/goroutes/definitions/notfound"
	"github.com/Nemutagk/goroutes/middlewares"
)

func LoadRoutes(list_routes []definitions.RouteGroup, server *http.ServeMux, dbConnectionsList map[string]db.DbConnection) *http.ServeMux {
	defaultMiddlewares := []definitions.Middleware{
		middlewares.InfoMiddleware,
		middlewares.CorsMiddleware,
		middlewares.AccessMiddleware,
		middlewares.MethodMiddleware,
	}

	globalRouteList := map[string]definitions.Route{}

	for _, groupRoute := range list_routes {
		routes := checkRouteGroup(groupRoute, "", defaultMiddlewares)

		for tmp_path, tmp_route := range routes {
			globalRouteList[tmp_path] = tmp_route
		}
	}

	keys := make([]string, 0, len(globalRouteList))
	for k := range globalRouteList {
		keys = append(keys, k)
	}
	slices.Sort(keys) // Sort routes by path
	info := ""
	for _, path := range keys {
		route := globalRouteList[path]

		info += fmt.Sprintf("\n%s:\t%s", route.Method, path)
		if len(*route.Middlewares) > 0 && goenvars.GetEnvBool("GOROUTES_DEBUG_MIDDLEWARES", false) {
			info += "\n    Middlewares:\n"
			for _, mw := range *route.Middlewares {
				fn := runtime.FuncForPC(reflect.ValueOf(mw).Pointer())
				name := "<unknown>"
				if fn != nil {
					name = fn.Name()
				}
				info += fmt.Sprintf("        %s\n", name)
			}
		}

		server.HandleFunc(path, applyMiddleware(route, dbConnectionsList))
	}

	if goenvars.GetEnvBool("GOROUTES_DEBUG", false) {
		golog.Log(context.Background(), "Routes loaded successfully", info)
	}

	return server
}

func AddNotFoundHandler(server *http.ServeMux, notFoundHandler http.HandlerFunc) http.HandlerFunc {
	return notfound.CustomMuxHandler(server, notFoundHandler)
}

func checkRouteGroup(routeGroup definitions.RouteGroup, parentPath string, parentMiddleware []definitions.Middleware) map[string]definitions.Route {
	path := preparePath(routeGroup.Prefix, parentPath)

	if routeGroup.Middlewares != nil && len(*routeGroup.Middlewares) > 0 {
		// log.Println("route group issues middleware!")
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
			golog.Error(context.Background(), "Invalid route definition:", route)
			continue
		}

		subPath := preparePath(routeDefine.Path, path)

		routeList = routeExists(routeList, subPath, routeDefine)

		for path, route := range routeList {
			if route.Middlewares == nil || len(*route.Middlewares) == 0 {
				tmpRoute := route
				listMiddlewares := make([]definitions.Middleware, 0)
				for _, mw := range parentMiddleware {
					if tmpRoute.ExcludeMiddlewares != nil && containsMiddleware(*tmpRoute.ExcludeMiddlewares, mw) {
						continue
					}
					listMiddlewares = append(listMiddlewares, mw)
				}

				tmpRoute.Middlewares = &listMiddlewares
				routeList[path] = tmpRoute
			} else {
				tmpRoute := route
				for _, pmw := range parentMiddleware {
					if !containsMiddleware(*tmpRoute.Middlewares, pmw) {
						if tmpRoute.ExcludeMiddlewares != nil && containsMiddleware(*tmpRoute.ExcludeMiddlewares, pmw) {
							continue
						}

						*tmpRoute.Middlewares = append([]definitions.Middleware{pmw}, *tmpRoute.Middlewares...)
					}
				}

				slices.Reverse(*tmpRoute.Middlewares)

				routeList[path] = tmpRoute
			}
		}
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
				golog.Error(context.Background(), "Route already exists:", route.Method, route.Path)
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
		for _, mw := range *route.Middlewares {
			route.Action = mw(route.Action, route, dbListConn)
		}

		return route.Action
	}

	subRoute, exists := route.Group[route.Method]
	if !exists {
		golog.Error(context.Background(), "No sub-route found for method:", route.Method, "in group:", route.Path)
		return func(w http.ResponseWriter, r *http.Request) {
			GoErrorResponse(w, *goerrors.NewGError("Method not allowed", goerrors.StatusMethodNotAllowed, nil, nil))
		}
	}

	for _, mw := range *subRoute.Middlewares {
		subRoute.Action = mw(subRoute.Action, subRoute, dbListConn)
	}

	return subRoute.Action
}

func GoErrorResponse(w http.ResponseWriter, err goerrors.GError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.GetStatusCode())
	w.Write([]byte(err.ToJson()))
}

// ToJson is an interface for types that can marshal themselves to JSON.
type ToJson interface {
	ToJson() []byte
}

func JsonResponse(w http.ResponseWriter, data any, statusCode int) {
	if data == nil {
		data = map[string]any{"message": "No content"}
	}

	var response []byte
	if v, ok := data.(ToJson); ok {
		response = v.ToJson()
	} else {
		var err error
		response, err = json.Marshal(data)
		if err != nil {
			golog.Error(context.Background(), "Error marshalling JSON response:", err)
			GoErrorResponse(w, *goerrors.NewGError("Failed to marshal JSON", goerrors.StatusInternalServerError, nil, goerrors.ConvertError(err)))
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(response)
}

func StringResponse(w http.ResponseWriter, data string, statusCode int, contentType *string) {
	if contentType == nil {
		defaultType := "text/plain"
		contentType = &defaultType
	}

	w.Header().Set("Content-Type", *contentType)
	w.WriteHeader(statusCode)
	w.Write([]byte(data))
}

func HttpResponse(w http.ResponseWriter, data string) {
	contentType := "text/html"
	StringResponse(w, data, http.StatusOK, &contentType)
}

func RawResponse(w http.ResponseWriter, data []byte, statusCode int, headers *map[string]string) {
	if headers == nil {
		headers = &map[string]string{
			"Content-Type": "application/octet-stream",
		}
	} else {
		if _, exists := (*headers)["Content-Type"]; !exists {
			(*headers)["Content-Type"] = "application/octet-stream"
		}
	}

	for key, value := range *headers {
		w.Header().Set(key, value)
	}

	w.WriteHeader(statusCode)
	w.Write(data)
}
