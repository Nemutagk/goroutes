package middlewares

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Nemutagk/goroutes/db"
	"github.com/Nemutagk/goroutes/definitions"
	"github.com/Nemutagk/goroutes/helper"

	"github.com/gofrs/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

func AccessMiddleware(next http.HandlerFunc, route definitions.Route) http.HandlerFunc {
	return func(wr http.ResponseWriter, r *http.Request) {
		fmt.Println("AccessMiddleware called")
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

		conn, err_conn := db.Connection()

		if err_conn != nil {
			fmt.Println("Error connecting to MongoDB: ", err_conn)
			wr.WriteHeader(http.StatusInternalServerError)
			wr.Write([]byte("Internal server error"))
			return
		}

		coll := conn.Database(helper.GetEnv("DB_LOGS_DATABASE", "kb_logs")).Collection("ip_black_list")
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

		collAccess := conn.Database(helper.GetEnv("DB_LOGS_DB", "kb_logs")).Collection("access")
		access_list, err_list := collAccess.Find(r.Context(), map[string]interface{}{
			"ip":            clientIp,
			"response_code": http.StatusNotFound,
			"created_at":    bson.M{"$gte": time.Now().Add(-10 * time.Minute)},
		})

		if err_list != nil {
			fmt.Println("Error finding access log:", err_list)
			wr.WriteHeader(http.StatusInternalServerError)
			wr.Write([]byte("Internal server error"))
			return
		}
		defer access_list.Close(r.Context())

		count := 0
		for access_list.Next(r.Context()) {
			count++
		}

		max_access_string := helper.GetEnv("MAX_ACCESS", "10")
		max_access, _ := strconv.Atoi(max_access_string)

		fmt.Println("Count of access logs in the last 10 minutes: " + strconv.Itoa(count) + ":" + max_access_string)

		if count > max_access {
			// IP is blacklisted for 1 hours
			_, err := coll.InsertOne(r.Context(), map[string]interface{}{
				"ip":         clientIp,
				"reason":     "Maximum number of 404 errors in less than " + max_access_string + " minutes",
				"expired_at": time.Now().Add(1 * time.Hour),
			})
			if err != nil {
				fmt.Println("Error inserting black list log:", err)
			}
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
			wr.WriteHeader(http.StatusForbidden)
			wr.Write([]byte("Access denied"))
			return
		}
		defer access_denied.Close(r.Context())

		count_denied_access := 0
		for access_denied.Next(r.Context()) {
			count_denied_access++
		}

		max_denied_acces_string := helper.GetEnv("MAX_DENIED_ACCESS", "3")
		max_denied_acces, _ := strconv.Atoi(max_denied_acces_string)
		if count_denied_access > max_denied_acces {
			// IP is blacklisted
			_, err := coll.InsertOne(r.Context(), map[string]interface{}{
				"ip":     clientIp,
				"reason": "Maximum number of 403 errors in less than " + max_denied_acces_string + " minutes",
			})

			if err != nil {
				fmt.Println("Error inserting black list log:", err)
			}
			wr.WriteHeader(http.StatusForbidden)
			wr.Write([]byte("Access denied"))
			return
		}

		//Registramos el acceso
		coll = conn.Database(helper.GetEnv("DB_LOGS_DB", "kb_logs")).Collection("access")
		body, newBody := mapBody(r.Body)
		r.Body = newBody
		_, err = coll.InsertOne(r.Context(), map[string]interface{}{
			"_id":        helper.GenerateUuid(),
			"app":        helper.GetEnv("APP_NAME", "sbframework"),
			"ip":         clientIp,
			"real_ip":    clientRealIp,
			"method":     route.Method,
			"path":       r.URL.Path,
			"body":       body,
			"header":     r.Header,
			"created_at": time.Now(),
			"updated_at": time.Now(),
		})

		if err != nil {
			fmt.Println("Error inserting access log:", err)
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
			fmt.Println("Error reading request body:", err)
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
