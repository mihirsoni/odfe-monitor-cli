package es

import (
	"bytes"
	"encoding/json"
	"net/http"
)

func MakeRequest(method string, endPoint string, body []byte, headers map[string]string) (map[string]interface{}, error) {
	var r map[string]interface{}
	client := http.Client{}
	req, err := http.NewRequest(method, endPoint, bytes.NewBuffer(body))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&r)
	return r, nil
}
