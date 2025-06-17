package middlewares

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Nemutagk/godb"
	"github.com/Nemutagk/godb/definitions/db"
	"github.com/Nemutagk/goenvars"
	"github.com/Nemutagk/goroutes/definitions"
	"github.com/Nemutagk/goroutes/helper"

	"github.com/gofrs/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

const ACCESS_CODE_ERROR = "0500"
const ACCESS_CODE_FORBIDDEN = "0403"
const ACCESS_CODE_DENIED = "0401"
const ACCESS_CODE_BLACKLISTED = "0404"
const ACCESS_CODE_NOT_FOUND = "0405"
const ACCESS_CODE_TOKEN_EXPIRED = "0406"

func AccessMiddleware(next http.HandlerFunc, route definitions.Route, dbListConn map[string]db.DbConnection) http.HandlerFunc {
	return func(wr http.ResponseWriter, r *http.Request) {
		log.Println("==================> AccessMiddleware called")
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

		if dbListConn == nil {
			log.Println("No database connection list provided")
			wr.WriteHeader(http.StatusInternalServerError)
			wr.Write([]byte("Internal server error"))
			return

		}

		db_conn_name := goenvars.GetEnv("DB_LOGS_CONNECTION", "mongodb")
		conn, err_con := godb.InitConnections(dbListConn).GetConnection(db_conn_name)

		if err_con != nil {
			log.Println("Error getting database connection:", err_con)
			wr.Header().Set("X-Request-Error", ACCESS_CODE_ERROR)
			wr.WriteHeader(http.StatusInternalServerError)
			wr.Write([]byte("Internal server error"))
			return
		}
		dbConn, _ := conn.ToMongoDb()

		coll := dbConn.Collection("ip_black_list")
		var result map[string]interface{}

		err := coll.FindOne(r.Context(), map[string]interface{}{
			"ip": clientIp,
			"$or": []bson.M{
				{"expired_at": bson.M{"$eq": nil}},
				{"expired_at": bson.M{"$lt": time.Now()}},
			},
		}).Decode(&result)

		if err == nil {
			// IP is blacklisted
			wr.WriteHeader(http.StatusForbidden)
			wr.Write([]byte("Access denied"))
			return
		}

		collAccess := dbConn.Collection("access")
		access_list, err_list := collAccess.Find(r.Context(), map[string]interface{}{
			"ip":            clientIp,
			"response_code": http.StatusNotFound,
			"created_at":    bson.M{"$gte": time.Now().Add(-10 * time.Minute)},
		})

		if err_list != nil {
			log.Println("Error finding access log:", err_list)
			// Error to find access log, return 500
			wr.Header().Set("X-Request-Error", ACCESS_CODE_ERROR)
			wr.WriteHeader(http.StatusInternalServerError)
			wr.Write([]byte("Internal server error"))
			return
		}
		defer access_list.Close(r.Context())

		count := 0
		for access_list.Next(r.Context()) {
			count++
		}

		max_access_string := goenvars.GetEnv("MAX_ACCESS", "10")
		max_access, _ := strconv.Atoi(max_access_string)

		log.Println("Count of access logs in the last 10 minutes: " + strconv.Itoa(count) + ":" + max_access_string)

		if count > max_access {
			// IP is blacklisted for 1 hours
			_, err := coll.InsertOne(r.Context(), map[string]interface{}{
				"ip":         clientIp,
				"reason":     "Maximum number of 404 errors in less than " + max_access_string + " minutes",
				"expired_at": time.Now().Add(1 * time.Hour),
			})
			if err != nil {
				log.Println("Error inserting black list log:", err)
			}
			wr.Header().Set("X-Request-Error", ACCESS_CODE_FORBIDDEN)
			wr.WriteHeader(http.StatusForbidden)
			wr.Write([]byte("Access denied"))
			return
		}

		access_denied, access_denied_err := collAccess.Find(r.Context(), map[string]interface{}{
			"ip":            clientIp,
			"response_code": http.StatusForbidden,
			"created_at":    bson.M{"$gte": time.Now().Add(-10 * time.Minute)},
		})

		if access_denied_err != nil {
			// IP is blacklisted
			wr.Header().Set("X-Request-Error", ACCESS_CODE_BLACKLISTED)
			wr.WriteHeader(http.StatusForbidden)
			wr.Write([]byte("Access denied"))
			return
		}
		defer access_denied.Close(r.Context())

		count_denied_access := 0
		for access_denied.Next(r.Context()) {
			count_denied_access++
		}

		max_denied_acces_string := goenvars.GetEnv("MAX_DENIED_ACCESS", "3")
		max_denied_acces, _ := strconv.Atoi(max_denied_acces_string)
		if count_denied_access > max_denied_acces {
			// IP is blacklisted
			_, err := coll.InsertOne(r.Context(), map[string]interface{}{
				"ip":     clientIp,
				"reason": "Maximum number of 403 errors in less than " + max_denied_acces_string + " minutes",
			})

			if err != nil {
				log.Println("Error inserting black list log:", err)
			}
			wr.Header().Set("X-Request-Error", ACCESS_CODE_FORBIDDEN)
			wr.WriteHeader(http.StatusForbidden)
			wr.Write([]byte("Access denied"))
			return
		}

		//Registramos el acceso
		coll = dbConn.Collection("access")
		body, newBody := mapBody(r.Body)
		r.Body = newBody

		body_save := map[string]interface{}{
			"_id":        helper.GenerateUuid(),
			"app":        goenvars.GetEnv("APP_NAME", "sbframework"),
			"ip":         clientIp,
			"real_ip":    clientRealIp,
			"method":     route.Method,
			"path":       r.URL.Path,
			"body":       body,
			"header":     r.Header,
			"created_at": time.Now(),
			"updated_at": time.Now(),
		}
		log.Println("request", body_save)

		_, err = coll.InsertOne(r.Context(), body_save)

		if err != nil {
			log.Println("Error inserting access log:", err)
			wr.WriteHeader(http.StatusInternalServerError)
			wr.Write([]byte("Internal server error"))
			return
		}

		// Generate unique request id with uuid v7
		type contextKey string
		requestIDKey := contextKey("request_id")
		ctx := context.WithValue(r.Context(), requestIDKey, uuid.Must(uuid.NewV7()).String())
		r = r.WithContext(ctx)

		// IP is not blacklisted, proceed with the request
		next(wr, r)
	}
}

func mapBody(raw_body io.ReadCloser) (map[string]interface{}, io.ReadCloser) {
	var body map[string]interface{}
	if raw_body != nil {
		// Leer el cuerpo de la solicitud
		bodyBytes, err := io.ReadAll(raw_body)
		if err != nil {
			log.Println("Error reading request body:", err)
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
	}

	return body, raw_body
}
