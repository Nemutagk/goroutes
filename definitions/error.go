package definitions

import "encoding/json"

type HttpError struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	Errors     string `json:"errors"`
	StatusCode int    `json:"status_code"`
}

func (e *HttpError) Error() string {
	return e.Message
}

func NewHttpError(message string, errors string, statusCode int) *HttpError {
	return &HttpError{
		Success:    false,
		Message:    message,
		Errors:     errors,
		StatusCode: statusCode,
	}
}
func (e *HttpError) ToJson() string {
	jsonData, err := json.Marshal(e)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func (e *HttpError) ToMap() map[string]interface{} {
	var errorDetail map[string]interface{}
	if err := json.Unmarshal([]byte(e.Errors), &errorDetail); err != nil {
		return map[string]interface{}{
			"success":     false,
			"message":     e.Message,
			"errors":      e.Errors,
			"status_code": e.StatusCode,
		}
	}

	return map[string]interface{}{
		"success":     false,
		"message":     e.Message,
		"errors":      errorDetail,
		"status_code": e.StatusCode,
	}
}
