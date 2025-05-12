package definitions

import (
	"context"
	"net/http"
)

type Middleware func(w http.HandlerFunc, route Route, cxt context.Context) http.HandlerFunc
