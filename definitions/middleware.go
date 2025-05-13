package definitions

import (
	"net/http"

	"github.com/Nemutagk/godb/definitions/db"
)

type Middleware func(w http.HandlerFunc, route Route, dbListConn map[string]db.DbConnection) http.HandlerFunc
