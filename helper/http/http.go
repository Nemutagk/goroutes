package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Nemutagk/goroutes/definitions"
)

func Response(res http.ResponseWriter, body any, statusCode int, contentType string) {
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	fmt.Println("body type:", fmt.Sprintf("%T", body))
	if contentType == "" {
		switch body.(type) {
		case string:
			contentType = "text/html"
		case []byte:
			contentType = "text/html"
		case map[string]interface{}:
			contentType = "application/json"
		case []map[string]interface{}:
			contentType = "application/json"
		default:
			contentType = "application/json"
		}
	}

	res.Header().Set("Content-Type", contentType)
	res.WriteHeader(statusCode)

	if contentType == "application/json" {
		json.NewEncoder(res).Encode(body)
	} else {
		res.Write([]byte(body.(string)))
	}
}

func ResponseError(res http.ResponseWriter, err error, message string) {
	var httpError *definitions.HttpError
	res.Header().Set("Content-Type", "application/json")

	if errors.As(err, &httpError) {
		res.WriteHeader(httpError.StatusCode)

		json.NewEncoder(res).Encode(httpError.ToMap())
		return
	}

	res.WriteHeader(http.StatusBadRequest)

	var errorDetail map[string]interface{}

	toError := map[string]interface{}{
		"success": false,
		"message": message,
		"errors":  nil,
	}

	if json.Unmarshal([]byte(err.Error()), &errorDetail) == nil {
		toError["errors"] = errorDetail
	} else {
		toError["errors"] = err.Error()
	}

	json.NewEncoder(res).Encode(toError)
}
