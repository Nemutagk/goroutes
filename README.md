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

Requiere Go 1.24 (ver [go.mod](go.mod)).

## Archivos y símbolos clave

- Carga y encadenado de rutas: [`goroutes.LoadRoutes`](routes.go) — [routes.go](routes.go)  
- Responses helpers: [`goroutes.JsonResponse`](routes.go), [`goroutes.StringResponse`](routes.go), [`goroutes.RawResponse`](routes.go), [`goroutes.GoErrorResponse`](routes.go) — [routes.go](routes.go)  
- Definiciones: [`definitions.Route`](definitions/route.go), [`definitions.RouteGroup`](definitions/route.go), [`definitions.Middleware`](definitions/middleware.go), [`definitions.HttpError`](definitions/error.go) — [definitions/](definitions/)  
- Middlewares incluidos:  
  - CORS: [`middlewares.CorsMiddleware`](middlewares/corsMiddleware.go) — [middlewares/corsMiddleware.go](middlewares/corsMiddleware.go)  
  - Acceso / rate & blacklist: [`middlewares.AccessMiddleware`](middlewares/accessMiddleware.go) — [middlewares/accessMiddleware.go](middlewares/accessMiddleware.go)  
  - Auth (ruta-por-ruta): [`middlewares.AuthMiddleware`](middlewares/authMiddleware.go) — [middlewares/authMiddleware.go](middlewares/authMiddleware.go)  
- Servicio de cuentas (validación token): [`service.AccountService`](service/accountService.go) — [service/accountService.go](service/accountService.go)  
- Not-found wrapper para mux: [`notfound.CustomMuxHandler`](definitions/notfound/notfound.go) — [definitions/notfound/notfound.go](definitions/notfound/notfound.go)  
- Utilidades: [`helper.GenerateUuid`](helper/helper.go), [`helper.PrettyPrint`](helper/helper.go) — [helper/helper.go](helper/helper.go)  
- Helpers HTTP alternativos: [`helper/http.Response`](helper/http/http.go), [`helper/http.ResponseError`](helper/http/http.go) — [helper/http/http.go](helper/http/http.go)  
- Response recorder utilitario: [`wr.NewResponseRecorder`](helper/http/wr/wr.go) — [helper/http/wr/wr.go](helper/http/wr/wr.go)

También revisa: [go.mod](go.mod) y [.devcontainer/devcontainer.json](.devcontainer/devcontainer.json).

## Cambios importantes reflejados en este README

1. Middlewares predeterminados en el cargador son CORS y Access (ver [`goroutes.LoadRoutes`](routes.go)). No existe un middleware `InfoMiddleware` ni `MethodMiddleware` en este workspace; referencias anteriores fueron removidas.
2. El handler de not-found expuesto es [`notfound.CustomMuxHandler`](definitions/notfound/notfound.go) — usa un ResponseRecorder para detectar rutas inexistentes y fallback.
3. La autenticación delegada hace una llamada HTTP con [`service.AccountService`](service/accountService.go). En caso de error HTTP devuelve un tipo `service.HTTPError`.
4. Logging/registro de accesos y blacklist se implementa en [`middlewares.AccessMiddleware`](middlewares/accessMiddleware.go) y requiere una conexión Mongo proporcionada a `LoadRoutes` (nombre de conexión por defecto desde `DB_LOGS_CONNECTION`).
5. El empaquetado de rutas admite grupos y agrupa métodos diferentes para la misma ruta (ver [`definitions.Route.Group`](definitions/route.go) y la lógica en [routes.go](routes.go)).

## Variables de entorno usadas (principales)

- ACCOUNT_API_URL — usado por [`service.AccountService`](service/accountService.go) (default: http://localhost:8080)  
- GOROUTES_DEBUG — controla impresión de rutas en [`goroutes.LoadRoutes`](routes.go)  
- GOROUTES_DEBUG_MIDDLEWARES — muestra middlewares por ruta en debug  
- DB_LOGS_CONNECTION — nombre de la conexión de logs en [`middlewares.AccessMiddleware`](middlewares/accessMiddleware.go)  
- MAX_ACCESS, MAX_DENIED_ACCESS, ACCESS_EXTRA_NODES_CENSORED, APP_NAME — control y censura en AccessMiddleware  
- CORS_ALLOW_* y CORS_EXPOSE_HEADERS, CORS_MAX_AGE, CORS_ALLOW_CREDENTIALS — usados por [`middlewares.CorsMiddleware`](middlewares/corsMiddleware.go)  
- GOROUTES_DISABLED_AWS_HEALTH_CHECKER — evita 200 automático para ELB health checks en [`applyMiddleware`](routes.go)

## Ejecución local mínima

Ejemplo mínimo de uso:

```go
package main

import (
  "net/http"
  "github.com/Nemutagk/goroutes"
  "github.com/Nemutagk/goroutes/definitions"
)

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
  // use CustomMuxHandler from package notfound if quiere custom 404:
  // http.ListenAndServe(":8080", notfound.CustomMuxHandler(mux, customNotFound))
  http.ListenAndServe(":8080", mux)
}
```

- Ejecutar local:  
  export GOROUTES_DEBUG=true  
  go run ./...

## Notas y recomendaciones

- Las funciones/documentación en este README se han actualizado para reflejar el código actual en [routes.go](routes.go), [middlewares/](middlewares/) y [service/accountService.go](service/accountService.go).  
- Si quieres añadir middlewares globales adicionales, pásalos en los grupos (`definitions.RouteGroup.Middlewares`) o en cada ruta (`definitions.Route.Middlewares`).  
- `ExcludeMiddlewares` permite remover para la ruta especificada un middleware global o middleware  grupal, está definido en `definitions.Route` pero su aplicación tiene lógica limitada; revisar [`addMiddleware`](routes.go) si necesitas exclusiones más específicas.  
- Mejoras sugeridas en el código: reusar http.Client en [`service.AccountService`](service/accountService.go), agregar timeouts/context a peticiones HTTP externas y tests unitarios.

## Licencia

Ver [LICENSE](LICENSE).