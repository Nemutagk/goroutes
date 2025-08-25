package middlewares

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Nemutagk/godb"
	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/goenvars"
	"github.com/Nemutagk/golog"
	"github.com/Nemutagk/goroutes/definitions"
	"github.com/Nemutagk/goroutes/helper"
	"github.com/Nemutagk/goroutes/helper/http/wr"
	"github.com/gofrs/uuid"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const ACCESS_CODE_ERROR = "0500"
const ACCESS_CODE_FORBIDDEN = "0403"
const ACCESS_CODE_DENIED = "0401"
const ACCESS_CODE_BLACKLISTED = "0404"
const ACCESS_CODE_NOT_FOUND = "0405"
const ACCESS_CODE_TOKEN_EXPIRED = "0406"

func AccessMiddleware(next http.HandlerFunc, route definitions.Route, dbListConn map[string]db.DbConnection) http.HandlerFunc {
	return func(res http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		golog.Log(ctx, "==================> AccessMiddleware called")
		clientIp, _ := getRealIp(r)

		if dbListConn == nil {
			golog.Error(ctx, "No database connection list provided")
			golog.Log(ctx, "==================> AccessMiddleware END")
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte("Internal server error"))
			return

		}

		dbConn := getConnection(ctx, dbListConn, res)
		if validateBlackList(ctx, dbConn, clientIp, res, r, route) {
			golog.Warning(ctx, "IP is blacklisted:", clientIp)
			golog.Log(ctx, "==================> AccessMiddleware END")
			res.Header().Set("X-Request-Error", ACCESS_CODE_BLACKLISTED)
			res.WriteHeader(http.StatusForbidden)
			res.Write([]byte("Access denied"))
			return
		}

		ctx = generateRequestId(ctx, r, clientIp, route)

		if validateRequest(ctx, dbConn, clientIp, res, r, route) {
			golog.Warning(ctx, "IP is blacklisted by request 401/403:", clientIp)
			golog.Log(ctx, "==================> AccessMiddleware END")
			res.Header().Set("X-Request-Error", ACCESS_CODE_FORBIDDEN)
			res.WriteHeader(http.StatusForbidden)
			res.Write([]byte("Access denied"))
			return
		}

		golog.Log(ctx, "==================> AccessMiddleware Medio")
		registerAccessLog(ctx, dbConn, res, r, route, http.StatusOK)

		wrEnv := wr.NewResponseRecorder(res)

		next(wrEnv, r)

		if wrEnv.GetStatus() != http.StatusOK {
			golog.Log(ctx, "==================> AccessMiddleware END")
			updateRequestStatus(dbConn, clientIp, wrEnv.GetStatus())
		}
	}
}

func mapBody(raw_body io.ReadCloser) (map[string]interface{}, io.ReadCloser) {
	var body map[string]interface{}
	if raw_body != nil {
		// Leer el cuerpo de la solicitud
		bodyBytes, err := io.ReadAll(raw_body)
		if err != nil {
			golog.Error(context.Background(), "Error reading request body:", err)
			return nil, raw_body
		}
		raw_body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		// Restaurar el cuerpo para que el controlador pueda usarlo
		// raw_body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Decodificar el cuerpo si es JSON
		if err := json.Unmarshal(bodyBytes, &body); err == nil {
			emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
			for key, value := range body {
				if strValue, ok := value.(string); ok {
					if key == "password" || key == "password_confirm" {
						body[key] = "*******"
					}

					if emailRegex.MatchString(strValue) {
						body[key] = strings.Split(strValue, "@")[0] + "@******+"
					}
				}
			}
		}

		extra_nodes_censored := goenvars.GetEnv("ACCESS_EXTRA_NODES_CENSORED", "")
		if extra_nodes_censored != "" {
			extra_nodes := strings.Split(extra_nodes_censored, ",")
			for _, node := range extra_nodes {
				if value, exists := body[node]; exists {
					if _, ok := value.(string); ok {
						body[node] = "*******"
					}
				}
			}
		}
	}

	return body, raw_body
}

func getRealIp(r *http.Request) (string, string) {
	clientRealIp := r.Header.Get("X-Forwarded-For")
	if clientRealIp == "" {
		clientRealIp = r.RemoteAddr
	}

	var clientIp = clientRealIp

	if strings.Contains(clientIp, ",") {
		clientIp = strings.Split(clientIp, ",")[0]
	}

	if strings.Contains(clientIp, ":") {
		clientIp = strings.Split(clientIp, ":")[0]
	}

	return clientIp, clientRealIp
}

func getConnection(ctx context.Context, dbListConn map[string]db.DbConnection, wr http.ResponseWriter) *mongo.Database {
	db_conn_name := goenvars.GetEnv("DB_LOGS_CONNECTION", "logs")
	conn, err_con := godb.InitConnections(dbListConn).GetConnection(db_conn_name)

	if err_con != nil {
		golog.Error(ctx, "Error getting database connection:", err_con)
		golog.Log(ctx, "==================> AccessMiddleware END")
		wr.Header().Set("X-Request-Error", ACCESS_CODE_ERROR)
		wr.WriteHeader(http.StatusInternalServerError)
		wr.Write([]byte("Internal server error"))
		return nil
	}
	dbConn, _ := conn.ToMongoDb()

	return dbConn
}

func generateRequestId(ctx context.Context, r *http.Request, clientIp string, route definitions.Route) context.Context {
	golog.Log(ctx, "Client IP:"+clientIp)
	golog.Log(ctx, "Route path:"+r.URL.String())
	golog.Log(ctx, "Route method:"+route.Method)

	requestId := uuid.Must(uuid.NewV7()).String()
	if rid := r.Header.Get("X-RequestKb-ID"); rid != "" {
		requestId = rid
	}

	ctx = context.WithValue(ctx, definitions.RequestIDKey, requestId)
	golog.Log(ctx, "Generated request ID:", requestId)

	return ctx
}

func registerAccessLog(ctx context.Context, dbConn *mongo.Database, wr http.ResponseWriter, r *http.Request, route definitions.Route, codeStatus int) {
	coll := dbConn.Collection("access")
	clientIp, clientRealIp := getRealIp(r)

	body, newBody := mapBody(r.Body)
	r.Body = newBody

	request_id := ctx.Value(definitions.RequestIDKey)
	if request_id == nil {
		request_id = "--"
	}

	body_save := map[string]interface{}{
		"_id":           helper.GenerateUuid(),
		"app":           goenvars.GetEnv("APP_NAME", "sbframework"),
		"ip":            clientIp,
		"real_ip":       clientRealIp,
		"method":        route.Method,
		"path":          GetFullRequestURL(r),
		"response_code": codeStatus,
		"body":          body,
		"header":        r.Header,
		"request_id":    request_id,
		"created_at":    time.Now(),
		"updated_at":    time.Now(),
	}

	golog.Log(ctx, "request", body_save)

	if _, err := coll.InsertOne(ctx, body_save); err != nil {
		golog.Error(ctx, "Error inserting access log:", err)
		wr.WriteHeader(http.StatusInternalServerError)
		wr.Write([]byte("Internal server error"))
	}
}

func validateBlackList(ctx context.Context, dbConn *mongo.Database, clientIp string, wr http.ResponseWriter, r *http.Request, route definitions.Route) bool {
	coll := dbConn.Collection("ip_black_list")

	exists, err := coll.CountDocuments(ctx, bson.M{
		"ip": clientIp,
		"$or": []bson.M{
			{"expired_at": bson.M{"$eq": nil}},
			{"expired_at": bson.M{"$lt": time.Now()}},
		},
	})

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) || errors.Is(err, mongo.ErrNilDocument) {
			return false // No blacklisted IP found
		}

		registerAccessLog(ctx, dbConn, wr, r, route, 403)
		golog.Error(ctx, "Error checking black list:", err)

		return true // Error occurred, treat as blacklisted
	}

	if exists > 0 {
		registerAccessLog(ctx, dbConn, wr, r, route, 403)
		golog.Error(ctx, "IP is blacklisted:", clientIp)
		return true // IP is blacklisted
	}

	return false
}

func addBlackList(dbConn *mongo.Database, clientIp string, expiredTime *time.Time) error {
	coll := dbConn.Collection("ip_black_list")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := coll.InsertOne(ctx, map[string]interface{}{
		"ip":         clientIp,
		"expired_at": expiredTime,
	})

	if err != nil {
		golog.Error(ctx, "Error inserting black list log:", err)
		return err
	}

	return nil
}

func validateRequest(ctx context.Context, dbConn *mongo.Database, clientIp string, wr http.ResponseWriter, r *http.Request, route definitions.Route) bool {
	collAccess := dbConn.Collection("access")
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	accessList, err := collAccess.Find(ctx, bson.M{
		"ip":         clientIp,
		"created_at": bson.M{"$gte": time.Now().Add(-1 * time.Hour)},
	})

	type AccessLog struct {
		Ip           string    `bson:"ip"`
		ResponseCode int       `bson:"response_code"`
		CreatedAt    time.Time `bson:"created_at"`
	}

	if err != nil {
		golog.Error(ctx, "Error finding access log:", err)
		registerAccessLog(ctx, dbConn, wr, r, route, 500)
		wr.Header().Set("X-Request-Error", ACCESS_CODE_ERROR)
		wr.WriteHeader(http.StatusInternalServerError)
		wr.Write([]byte("Internal server error"))
		return true // Error occurred
	}
	defer accessList.Close(ctx)

	count403Request := 0
	count401Request := 0

	for accessList.Next(ctx) {
		var accessLog AccessLog
		if err := accessList.Decode(&accessLog); err != nil {
			golog.Error(ctx, "Error decoding access log:", err)
			continue
		}

		if accessLog.ResponseCode == http.StatusForbidden && accessLog.CreatedAt.After(time.Now().Add(-10*time.Minute)) {
			count403Request++
		} else if accessLog.ResponseCode == http.StatusUnauthorized && accessLog.CreatedAt.After(time.Now().Add(-10*time.Minute)) {
			count401Request++
		}
	}

	if goenvars.GetEnvInt("MAX_DENIED_ACCESS", 3) < count403Request {
		// IP is blacklisted
		exp := time.Now().Add((24 * 365) * time.Hour)
		addBlackList(dbConn, clientIp, &exp)
		registerAccessLog(ctx, dbConn, wr, r, route, 403)
		return true
	}

	if goenvars.GetEnvInt("MAX_ACCESS", 10) < count401Request {
		// IP is blacklisted
		exp := time.Now().Add(24 * time.Hour)
		addBlackList(dbConn, clientIp, &exp)
		registerAccessLog(ctx, dbConn, wr, r, route, 401)
		return true
	}

	return false
}

func updateRequestStatus(dbConn *mongo.Database, clientIp string, status int) error {
	coll := dbConn.Collection("access")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	request_id := ctx.Value("request_id")
	if request_id == nil {
		request_id = "--"
	}

	_, err := coll.UpdateMany(ctx, bson.M{
		"ip":         clientIp,
		"request_id": request_id,
	}, bson.M{"$set": bson.M{"response_code": status}})
	if err != nil {
		golog.Error(ctx, "Error updating access log:", err)
		return err
	}

	return nil
}

func GetFullRequestURL(r *http.Request) string {
	// Protocolo
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if r.URL.Scheme != "" {
			proto = r.URL.Scheme
		} else if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}

	// Host
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}

	// Puerto
	port := r.Header.Get("X-Forwarded-Port")
	// Si el host ya incluye puerto, no lo agregamos
	if port != "" && !strings.Contains(host, ":") {
		host = host + ":" + port
	}

	// Path y query
	uri := r.Header.Get("X-Forwarded-Uri")
	if uri == "" {
		uri = r.RequestURI
	}

	// Construir URL completa
	return proto + "://" + host + uri
}
