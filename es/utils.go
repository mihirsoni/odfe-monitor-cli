package es

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Response struct {
	Status int
	Data   map[string]interface{}
}

func MakeRequest(method string,
	endPoint string,
	body []byte,
	headers map[string]string) (Response, error) {
	var response Response
	client := http.Client{}
	req, err := http.NewRequest(method, endPoint, bytes.NewBuffer(body))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&response.Data)
	response.Status = resp.StatusCode
	return response, nil
}
