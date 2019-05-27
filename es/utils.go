package es

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

type Response struct {
	Status int
	Data   map[string]interface{}
}

var httpClient *retryablehttp.Client

func init() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient = retryablehttp.NewClient()
	httpClient.HTTPClient.Transport = tr
	httpClient.RetryWaitMin = 200 * time.Millisecond
	httpClient.CheckRetry = checkRetry
	httpClient.Logger = nil
}

func checkRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// Handling special bad request of resource creation else relying on default policy for 400
	// Don't retry 400 for all bad request
	if resp.StatusCode == 400 {
		var data map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&data)
		// This is required to reassign the Body as we're reading it. Stream can't be read twice
		dataByte, _ := json.Marshal(data)
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(dataByte))
		var reason string
		respErr := data["error"].(map[string]interface{})
		if respErr != nil {
			reason = data["error"].(map[string]interface{})["type"].(string)
		}
		if reason == "resource_already_exists_exception" {
			return true, nil
		}
	}
	return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
}

// MakeRequest initiate request to ES API
func (esClient *Client) MakeRequest(method string,
	endPoint string,
	body []byte,
	headers map[string]string) (Response, error) {
	var response Response
	var err error
	req, err := retryablehttp.NewRequest(method, esClient.URL+endPoint, bytes.NewBuffer(body))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	req.SetBasicAuth(esClient.Username, esClient.Password)
	doneCh := make(chan bool)
	go func() {
		defer close(doneCh)
		resp, err := httpClient.Do(req)
		if err != nil {
			doneCh <- false
			return
		}
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&response.Data)
		response.Status = resp.StatusCode
		doneCh <- true
	}()
	if <-doneCh {
		return response, nil
	}
	return response, err

}
