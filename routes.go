package goroutes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/goenvars"
	"github.com/Nemutagk/goerrors"
	"github.com/Nemutagk/golog"
	"github.com/Nemutagk/goroutes/definitions"
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

	for _, gr := range list_routes {
		tmpRoutes := checkRoute(gr, "/", defaultMiddlewares)
		for path, route := range tmpRoutes {
			if _, ok := globalRouteList[path]; ok {
				golog.Error(context.Background(), "Route already exists:", path, "Method:", route.Method)
				continue
			}

			globalRouteList[path] = route
		}
	}

	if goenvars.GetEnvBool("GOROUTES_DEBUG", false) {
		showRoutesExists(globalRouteList)
	}

	for path, route := range globalRouteList {
		// log.Println("Registering route:", route.Path, "Method:", route.Method)
		server.HandleFunc(path, applyMiddleware(route, dbConnectionsList))
	}

	return server
}

func checkRoute(rg definitions.RouteGroup, parentPath string, parentMiddleware []definitions.Middleware) map[string]definitions.Route {
	basePath := preparePath(rg.Prefix, parentPath)

	// Agregamos los middlewares del grupo padre
	if rg.Middlewares != nil && len(*rg.Middlewares) > 0 {
		for _, md := range *rg.Middlewares {
			if !containsMiddleware(parentMiddleware, md) {
				parentMiddleware = append(parentMiddleware, md)
			}
		}
	}

	allRoutes := make(map[string]definitions.Route)

	// Listamos todas las rutas del grupo
	for _, route := range rg.Routes {
		// validamos si la ruta a checar es otro grupo (subgrupo)
		if subroute, ok := route.(definitions.RouteGroup); ok {
			// si es un subgrupo, llamamos recursivamente a checkRoute
			tmpRoutes := checkRoute(subroute, basePath, parentMiddleware)
			// agregamos las rutas del subgrupo a la lista de rutas
			for path, subRoute := range tmpRoutes {
				allRoutes[path] = subRoute
			}

			continue
		}

		// si no es un subgrupo, validamos que sea una ruta
		routeDef, ok := route.(definitions.Route)
		if !ok {
			golog.Error(context.Background(), "Invalid route definition:", route)
			continue
		}

		// Validamos que la ruta tenga un path definido y que no sea un path repetido o vacio
		// si el path existe se genera un grupo dentro de la ruta donde se resguardan los motodos,
		// esta pensando para una api restfull donde los metodos pueden diferir de una ruta aunque sea
		// textualmente la misma
		allRoutes = routeExists(allRoutes, basePath, routeDef)
	}

	for path, route := range allRoutes {
		// Verificamos que las rutas tengan los middlewares del grupo padre incluidos
		// middlewares que deben cargar por default
		if route.Group == nil {
			allRoutes[path] = addMiddleware(route, parentMiddleware)
			continue
		}

		// Si la ruta tiene un grupo, agregamos los middlewares del grupo padre a cada subruta
		for method, subRoute := range route.Group {
			route.Group[method] = addMiddleware(subRoute, parentMiddleware)
		}

		allRoutes[path] = route
	}

	return allRoutes
}

func addMiddleware(route definitions.Route, parentMiddleware []definitions.Middleware) definitions.Route {
	// No hay middlewares definidos en la ruta, se agregan los del padre
	if route.Middlewares == nil || len(*route.Middlewares) == 0 {
		//si no hay exclusión de middlewares, se agregan todos los del padre
		if route.ExcludeMiddlewares == nil || len(*route.ExcludeMiddlewares) == 0 {
			route.Middlewares = &parentMiddleware
			return route
		}

		// Si hay exclusión de middlewares, se agregan los del padre menos los excluidos
		for _, emd := range *route.ExcludeMiddlewares {
			if !containsMiddleware(parentMiddleware, emd) {
				tmpRoutes := append(*route.Middlewares, emd)
				route.Middlewares = &tmpRoutes
			}
		}
	} else {
		// Si ya hay middlewares definidos en la ruta, se agregan los del padre menos los excluidos
		// ni tampoco los que ya están definidos en la ruta
		mdPreparados := []definitions.Middleware{}
		for _, md := range parentMiddleware {
			if route.ExcludeMiddlewares == nil || len(*route.ExcludeMiddlewares) == 0 {
				if !containsMiddleware(*route.Middlewares, md) {
					mdPreparados = append(mdPreparados, md)
				}

				continue
			}

			for _, emd := range *route.ExcludeMiddlewares {
				if !containsMiddleware(*route.Middlewares, emd) {
					mdPreparados = append(mdPreparados, emd)
				}
			}
		}

		for _, mdP := range mdPreparados {
			if !containsMiddleware(*route.Middlewares, mdP) {
				tmpMd := append(*route.Middlewares, mdP)
				route.Middlewares = &tmpMd
			}
		}
	}

	return route
}

func routeExists(routeList map[string]definitions.Route, parentPath string, route definitions.Route) map[string]definitions.Route {
	//generamos la ruta completa a partir del prefijo y el path del padre
	path := preparePath(route.Path, parentPath)

	// si es una ruta que no existe en el grupo global, la agregamos
	if _, exists := routeList[path]; !exists {
		routeList[path] = route
		return routeList
	}

	// si la ruta ya existe extraemos la ruta original
	orginalRoute := routeList[path]

	// si la ruta original no tiene un grupo, lo creamos
	// y agregamos la ruta original al grupo con el método correspondiente
	if orginalRoute.Group == nil {
		orginalRoute.Group = map[string]definitions.Route{}
		orginalRoute.Group[orginalRoute.Method] = orginalRoute
	}

	// verificamos que la nueva ruta no tenga un método ya registrado en el grupo
	if _, exists := route.Group[route.Method]; exists {
		golog.Error(context.Background(), "Route already exists for path:", parentPath, "Method:", route.Method)
		return routeList
	}

	// agregamos la nueva ruta al grupo de la ruta original
	// y agregamos la ruta original actualizada al grupo global
	orginalRoute.Group[route.Method] = route
	routeList[path] = orginalRoute

	return routeList
}

func preparePath(prefix string, parentPath string) string {
	// si parentPath es raiz "/" y prefix comienza con "/", lo eliminamos
	if parentPath == "/" {
		if prefix != "" && prefix[0:1] == "/" {
			prefix = prefix[1:]
		}
	}

	// concatenamos el parentPath con el prefix
	path := parentPath + prefix

	// verificamos que si la ruta es mayor a 0 caracteres validamos si el último
	// caracter es una barra, si es así la eliminamos
	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	// si la ruta es menor a 1 caracter, la convertimos en "/"
	if path == "" {
		path = "/"
	}

	return path
}

func containsMiddleware(middleware []definitions.Middleware, mw definitions.Middleware) bool {
	// Recorremos los middlewares de la lista y comparamos con el middleware dado
	for _, m := range middleware {
		if fmt.Sprintf("%p", m) == fmt.Sprintf("%p", mw) {
			// Si encontramos un middleware que coincide, retornamos true
			return true
		}
	}

	// Si no encontramos coincidencias, retornamos false
	return false
}

func applyMiddleware(route definitions.Route, dbListConn map[string]db.DbConnection) http.HandlerFunc {
	// si la ruta no tiene grupo ejecutamos retornamos la acción directamente
	if route.Group == nil || len(route.Group) == 0 {
		if route.Middlewares != nil {
			// aplicamos los middlewares a la acción de la ruta
			for _, mw := range *route.Middlewares {
				route.Action = mw(route.Action, route, dbListConn)
			}
		}

		return route.Action
	}

	// si la ruta tiene un grupo, buscamos el subgrupo correspondiente al método de la ruta que
	// se está ejecutando
	subRoute, exists := route.Group[route.Method]
	if !exists {
		golog.Error(context.Background(), "No sub-route found for method:", route.Method, "in group:", route.Path)
		return func(w http.ResponseWriter, r *http.Request) {
			GoErrorResponse(w, *goerrors.NewGError("Method not allowed", goerrors.StatusMethodNotAllowed, nil, nil))
		}
	}

	// aplicamos los middlewares a la acción del subgrupo
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

func showRoutesExists(routeList map[string]definitions.Route) {
	all_routes := []string{}

	keys := make([]string, 0, len(routeList))
	for k := range routeList {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	txtInfo := "======================================================\n" + "Registered routes:\n"
	for _, path := range keys {
		route := routeList[path]
		if route.Group == nil {
			txtInfo += getInfoRoute(route, path)

			continue
		}

		for _, subRoute := range route.Group {
			txtInfo += getInfoRoute(subRoute, path)
		}
	}

	txtInfo += "Total routes registered: " + fmt.Sprint(len(all_routes)) + "\n"
	txtInfo += "======================================================\n"
	fmt.Println(txtInfo)
}

func getInfoRoute(route definitions.Route, path string) string {
	txtInfo := route.Method + "\t" + path + "\n"

	if goenvars.GetEnvBool("GOROUTES_DEBUG_MIDDLEWARES", false) {
		if route.Middlewares != nil && len(*route.Middlewares) > 0 {
			txtInfo += fmt.Sprintf("\tMiddlewares (%d):\n", len(*route.Middlewares))
			for _, mw := range *route.Middlewares {
				txtInfo += "\t\t" + funcName(mw) + "\n"
			}
			txtInfo += "\n"
		} else {
			txtInfo += "Middlewares: None\n"
		}
	}

	return txtInfo
}

func funcName(fn any) string {
	if fn == nil {
		return "<nil>"
	}
	val := reflect.ValueOf(fn)
	if !val.IsValid() {
		return "<invalid>"
	}
	pc := val.Pointer()
	if f := runtime.FuncForPC(pc); f != nil {
		name := f.Name()
		// Quitar path completo y paquete
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		// Quitar paquete dejando solo el identificador final
		if i := strings.LastIndex(name, "."); i >= 0 {
			name = name[i+1:]
		}
		return name
	}
	return "<unknown>"
}
