package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Nemutagk/goroutes/helper"
)

type HTTPError struct {
	Code   int
	Status string
	Body   []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Code, e.Status)
}

func AccountService(path, method string, payload interface{}) (any, error) {
	baseUrl := helper.GetEnv("ACCOUNT_SERVICE_URL", "http://localhost:8080")

	lastLetterBaseUrl := baseUrl[len(baseUrl)-1:]
	if lastLetterBaseUrl == "/" {
		baseUrl = baseUrl[:len(baseUrl)-1]
	}

	firstLetterPath := path[0:1]
	if firstLetterPath != "/" {
		path = "/" + path
	}

	lastLetterPath := path[len(path)-1:]
	if lastLetterPath == "/" {
		path = path[:len(path)-1]
	}

	url := baseUrl + path
	var requestBody []byte
	var err error

	if payload != nil {
		requestBody, err = json.Marshal(payload)
		if err != nil {
			fmt.Println("Error marshalling payload:", err)
			return nil, err
		}
	}

	fmt.Println("Request URL:", url)
	fmt.Println("Request Method:", method)

	req, err := http.NewRequest(method, url, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Println("Error response from server:", resp.Status)
		return nil, &HTTPError{
			Code:   resp.StatusCode,
			Status: resp.Status,
			Body:   body,
		}
	}

	var result map[string]any

	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println("Error decoding response:", err)
		return nil, err
	}

	return result, nil
}
