package goroutes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/goenvars"
	"github.com/Nemutagk/goerrors"
	"github.com/Nemutagk/golog"
	"github.com/Nemutagk/goroutes/definitions"
	"github.com/Nemutagk/goroutes/definitions/notfound"
	"github.com/Nemutagk/goroutes/helper"
	"github.com/Nemutagk/goroutes/middlewares"
)

func LoadRoutes(list_routes []definitions.RouteGroup, server *http.ServeMux, dbConnectionsList map[string]db.DbConnection) *http.ServeMux {
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

	if goenvars.GetEnvBool("GOROUTES_DEBUG", false) {
		golog.Log(context.Background(), "Routes loaded successfully")
		helper.PrettyPrint(totalRouteList)
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

		if len(parentMiddleware) > 0 {
			if routeDefine.Middlewares == nil {
				routeDefine.Middlewares = &parentMiddleware
			} else {
				// mws := append(*routeDefine.Middlewares, parentMiddleware...)
				if !containsMiddleware(*routeDefine.Middlewares, middlewares.InfoMiddleware) {
					mvs := append(*routeDefine.Middlewares, middlewares.InfoMiddleware)
					routeDefine.Middlewares = &mvs
				}
				if !containsMiddleware(*routeDefine.Middlewares, middlewares.MethodMiddleware) {
					mvs := append(*routeDefine.Middlewares, middlewares.MethodMiddleware)
					routeDefine.Middlewares = &mvs
				}
				if !containsMiddleware(*routeDefine.Middlewares, middlewares.CorsMiddleware) {
					mvs := append(*routeDefine.Middlewares, middlewares.CorsMiddleware)
					routeDefine.Middlewares = &mvs
				}
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
	// Queremos que InfoMiddleware sea el primero que se EJECUTE (outermost)
	defaultMiddlewares := []definitions.Middleware{
		middlewares.InfoMiddleware,
		middlewares.CorsMiddleware,
		middlewares.MethodMiddleware,
		middlewares.AccessMiddleware,
	}

	buildChain := func(rt definitions.Route) http.HandlerFunc {
		var finalList []definitions.Middleware
		if rt.Middlewares == nil {
			finalList = append(finalList, defaultMiddlewares...)
		} else {
			// Copiamos manteniendo orden existente
			finalList = append(finalList, *rt.Middlewares...)
			// Garantizamos que InfoMiddleware esté presente y al inicio
			if !containsMiddleware(finalList, middlewares.InfoMiddleware) {
				finalList = append([]definitions.Middleware{middlewares.InfoMiddleware}, finalList...)
			} else {
				// Si estaba pero no en posición 0 lo movemos
				if fmt.Sprintf("%p", finalList[0]) != fmt.Sprintf("%p", middlewares.InfoMiddleware) {
					tmp := []definitions.Middleware{middlewares.InfoMiddleware}
					for _, mw := range finalList {
						if fmt.Sprintf("%p", mw) != fmt.Sprintf("%p", middlewares.InfoMiddleware) {
							tmp = append(tmp, mw)
						}
					}
					finalList = tmp
				}
			}
			// Añadimos los que falten (excepto Info ya tratado)
			for _, dmw := range defaultMiddlewares {
				if !containsMiddleware(finalList, dmw) {
					finalList = append(finalList, dmw)
				}
			}
		}

		// Construcción de cadena: índice 0 se ejecuta primero (envolvemos en orden inverso)
		handler := rt.Action
		for i := len(finalList) - 1; i >= 0; i-- {
			handler = finalList[i](handler, rt, dbListConn)
		}
		return handler
	}

	if route.Group == nil {
		return buildChain(route)
	}

	return func(res http.ResponseWriter, req *http.Request) {
		if subRoute, exists := route.Group[req.Method]; exists {
			handler := buildChain(subRoute)
			handler(res, req)
			return
		}

		golog.Error(req.Context(), "Route not found:", req.Method, req.URL.Path)
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusNotFound)
		jsonResponse, err := json.Marshal(map[string]string{"error": "Method not supported"})
		if err != nil {
			golog.Error(req.Context(), "Error marshalling JSON response:", err)
			GoErrorResponse(res, *goerrors.NewGError("Failed to marshal JSON", goerrors.StatusInternalServerError, nil, goerrors.ConvertError(err)))
			return
		}
		res.Write(jsonResponse)
	}
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
