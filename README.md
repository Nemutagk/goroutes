# goroutes

Librería ligera para definir y montar rutas HTTP en Go con soporte para:
- Grupos de rutas y prefijos
- Middlewares encadenables (orden controlado)
- Autenticación delegada vía API externa
- Registro estructurado de accesos con MongoDB
- Manejo centralizado de respuestas JSON / texto / raw
- Handler personalizado para rutas no encontradas (404)

## Instalación

```bash
go get github.com/Nemutagk/goroutes
```

Requiere Go 1.21+ (el `go.mod` usa `toolchain go1.24.5`).

## Archivos clave

- Carga y encadenado de rutas: [`goroutes.LoadRoutes`](routes.go) en [routes.go](routes.go)
- Respuestas HTTP helpers: [`goroutes.JsonResponse`](routes.go), [`goroutes.StringResponse`](routes.go), [`goroutes.RawResponse`](routes.go)
- Manejo de errores GoError: [`goroutes.GoErrorResponse`](routes.go)
- Definiciones: [definitions/route.go](definitions/route.go), [definitions/middleware.go](definitions/middleware.go), [definitions/error.go](definitions/error.go)
- Middlewares:  
  - Info: [`middlewares.InfoMiddleware`](middlewares/infoMiddleware.go)  
  - CORS: [`middlewares.CorsMiddleware`](middlewares/corsMiddleware.go)  
  - Método: [`middlewares.MethodMiddleware`](middlewares/methodMiddleware.go)  
  - Acceso / rate & blacklist: [`middlewares.AccessMiddleware`](middlewares/accessMiddleware.go)  
  - Auth externo: [`middlewares.AuthMiddleware`](middlewares/authMiddleware.go)  
  - Lista base: [middlewares/list/list.go](middlewares/list/list.go)
- Servicio de cuentas (validación token): [`service.AccountService`](service/accountService.go)
- 404 personalizado: [`notfound.CustomMuxHandler`](definitions/notfound/notfound.go)
- Utilidades: [helper/helper.go](helper/helper.go), [helper/http/http.go](helper/http/http.go)

## Conceptos

### Definición de rutas

```go
import (
  "net/http"
  "github.com/Nemutagk/goroutes/definitions"
  "github.com/Nemutagk/goroutes"
)

var api = definitions.RouteGroup{
  Prefix: "/api",
  Routes: []interface{}{
    definitions.Route{
      Path:   "/ping",
      Method: http.MethodGet,
      Action: func(w http.ResponseWriter, r *http.Request) {
        goroutes.JsonResponse(w, map[string]any{"pong": true}, http.StatusOK)
      },
    },
  },
}
```

### Grupos anidados y middlewares

```go
routes := definitios.RouteGroup{
  Prefix: "/api",
  Routes: []interface{}{
    routesV1,
  },
}

routesV1 = definitions.RouteGroup{
  Prefix: "/v1",
  Routes: []interface{}{
    definitions.Route{
      Path:   "/public",
      Method: http.MethodGet,
      Action: publicHandler,
    },
    definitions.Route{
      Path:   "/secure",
      Method: http.MethodGet,
      Auth:   &definitions.RouteAuth{App: "core", Permission: "read.secure"},
      Action: secureHandler,
    },
  },
}

anotherRoute := definitions.RouteGroup{
  Prefix: "/annother",
  Routes: []interface{}{
    definitions.Route{
      Path: "/",
      Method: "GET",
      Action: func(w http.ResponseWriter, r *http.Request) {
        goroutes.JsonResponse(w, map[string]interface{"success":true,"message":"Welcome to GoRoutes"})
      }
    },
    definitions.Route{
      Path: "/{id}",
      Method: "GET",
      Action: func(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        goroutes.JsonResponse(w, map[string]interface{"success":true,"message":"Return item by id ("+id+")"})
      }
    },
    definitions.Route{
      Path: "/",
      Method: "POST",
      Action: func(w http.ResponseWriter, r *http.Request) {
        goroutes.JsonResponse(w, map[string]interface{"success":true,"message":"Post endpoint"})
      }
    },
  }
}
```

### Carga de rutas

```go
mux := http.NewServeMux()
goroutes.LoadRoutes(routes, mux, dbConnections) // dbConnections: map[string]db.DbConnection
http.ListenAndServe(":8080", goroutes.AddNotFoundHandler(mux, customNotFound))
```

### Not Found personalizado

```go
customNotFound := func(w http.ResponseWriter, r *http.Request) {
  goroutes.JsonResponse(w, map[string]any{"error": "route not found"}, http.StatusNotFound)
}
```

### Middleware y orden

Orden final (si no defines otros):  
1. Info  
2. CORS  
3. Method  
4. Access  
+ (Auth si la ruta define `Auth` porque está antes en la lista base manual que construyas)

`LoadRoutes` garantiza que [`middlewares.InfoMiddleware`](middlewares/infoMiddleware.go) vaya primero. Puedes añadir otros con `Middlewares: &[]definitions.Middleware{ ... }`.

### Autenticación

Si la ruta incluye `Auth: &definitions.RouteAuth{App: "...", Permission: "..."}` se dispara [`middlewares.AuthMiddleware`](middlewares/authMiddleware.go), que llama a [`service.AccountService`](service/accountService.go) contra `/auth/validation` en `ACCOUNT_API_URL`.

### Respuestas

Usa helpers:
- JSON: `goroutes.JsonResponse(w, data, status)`
- Texto: `goroutes.StringResponse(w, "ok", 200, nil)`
- HTML: `goroutes.HttpResponse(w, "<h1>Hi</h1>")`
- Binario: `goroutes.RawResponse(w, bytes, 200, &headers)`

### Manejo de errores

Para errores de negocio puedes usar `goerrors` (si integras) y enviar con `goroutes.GoErrorResponse`.

### Registro de acceso

[`middlewares.AccessMiddleware`](middlewares/accessMiddleware.go):
- Guarda documento en colección `access`
- Aplica límites de 404/403 y blacklist temporal en `ip_black_list`
- Censura campos sensibles (password, emails, campos extra vía env)

Requiere conexión Mongo (vía `godb`):
`dbConnections map[string]db.DbConnection` debe contener la conexión cuyo nombre coincide con `DB_LOGS_CONNECTION`.

## Variables de entorno

| Variable | Default | Descripción |
|----------|---------|-------------|
| ACCOUNT_API_URL | http://localhost:8080 | Endpoint servicio de cuentas |
| GOROUTES_DEBUG | false | Muestra rutas cargadas |
| DB_LOGS_CONNECTION | logs | Nombre de conexión usada para logs |
| MAX_ACCESS | 10 | Máx. 404 en 10 min antes de blacklist temporal |
| MAX_DENIED_ACCESS | 3 | Máx. 403 en 10 min |
| ACCESS_EXTRA_NODES_CENSORED | "" | Campos extra a censurar en body (comma separated) |
| APP_NAME | sbframework | Nombre que se guarda en logs |
| MAX_DENIED_ACCESS | 3 | Límite 403 |
| (Auth header) | Authorization | Token enviado al validar |

## Servicio de cuentas

`AccountService(path, method, payload)` hace request JSON y devuelve `map[string]any` o `HTTPError`.

Ejemplo manual:
```go
res, err := service.AccountService("/auth/validation", "POST", map[string]string{
  "token": "Bearer abc",
  "app": "core",
  "permission": "read.secure",
})
```

## Extender con middlewares propios

```go
myMw := func(next http.HandlerFunc, rt definitions.Route, db map[string]db.DbConnection) http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
    // pre
    next(w, r)
    // post
  }
}

definitions.Route{
  Path: "/x",
  Method: http.MethodGet,
  Middlewares: &[]definitions.Middleware{myMw},
  Action: h,
}
```

Si quieres excluir alguno por ruta puedes extender el patrón añadiendo lógica sobre `ExcludeMiddlewares` (ya existe campo en `definitions.Route`, pero aún no se procesa; puedes modificar `applyMiddleware`).

## 404 / Métodos

Si un path existe pero método no, el wrapper responde JSON: `{"error":"Method not supported"}`.

## Ejecución local

```bash
export GOROUTES_DEBUG=true
go run ./...
```

## Ejemplo mínimo

```go
func main() {
  mux := http.NewServeMux()
  routes := []definitions.RouteGroup{
    { Prefix: "/api", Routes: []interface{}{
      definitions.Route{
        Path: "/ping", Method: http.MethodGet,
        Action: func(w http.ResponseWriter, r *http.Request) {
          goroutes.JsonResponse(w, map[string]any{"pong": true}, http.StatusOK)
        },
      },
    }},
  }
  goroutes.LoadRoutes(routes, mux, nil)
  http.ListenAndServe(":8080", goroutes.AddNotFoundHandler(mux, func(w http.ResponseWriter, r *http.Request){
    goroutes.JsonResponse(w, map[string]any{"error":"not found"}, http.StatusNotFound)
  }))
}
```

## Buenas prácticas

- Evita rutas duplicadas; el sistema agrupa métodos en misma `Path`.
- Usa siempre `http.MethodGet` etc. para consistencia.
- Centraliza creación de conexiones y pásalas a `LoadRoutes`.
- Para ocultar símbolos sensibles añade claves a `ACCESS_EXTRA_NODES_CENSORED`.

## Roadmap sugerido

- Implementar soporte efectivo de `ExcludeMiddlewares`
- Tiempo de expiración configurable para blacklist
- Métricas Prometheus
- Pool de http.Client reutilizable en `AccountService`
- Context timeout en peticiones externas

## Licencia

MIT (ajusta según corresponda).