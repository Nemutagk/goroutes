package definitions

import "net/http"

type Middleware func(w http.HandlerFunc, route Route) http.HandlerFunc
