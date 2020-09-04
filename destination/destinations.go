/*
 * Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License").
 * You may not use this file except in compliance with the License.
 * A copy of the License is located at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * or in the "license" file accompanying this file. This file is distributed
 * on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
 * express or implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

package destination

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mihirsoni/odfe-monitor-cli/es"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

//FileName where destinations are stored and read from
const FileName = "destinations.yaml"
const indexSearchURL = "/.opendistro-alerting-config/_search"

//GetLocal Parse local destinations
func GetLocal(rootDir string) (map[string]string, error) {
	destinations := make(map[string]string)
	destinationsPath := filepath.Join(rootDir, FileName)
	if _, err := os.Stat(destinationsPath); os.IsNotExist(err) {
		return nil, errors.Wrap(err, FileName+"doesn't exist")
	}
	yamlFile, err := ioutil.ReadFile(destinationsPath)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to read"+FileName)
	}
	yaml.Unmarshal(yamlFile, &destinations)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to parse "+FileName)
	}
	return destinations, nil
}

func getCommonHeaders() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

// GetRemote This will get all the monitor and write them into destinations.yaml file on the root directory
func GetRemote(esClient es.Client) (map[string]Destination, error) {
	// Adding 10k which will not be the case.
	getAllDestinationQuery := []byte(`{"size": 10000, "query":{ "bool": {"must": { "exists": { "field" : "destination" }}}}}`)
	resp, err := esClient.MakeRequest(http.MethodPost,
		indexSearchURL,
		getAllDestinationQuery,
		getCommonHeaders(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to fetch destinations from elasticsearch")
	}
	allRemoteDestinationsMap := make(map[string]Destination)
	if resp.Status == 200 {
		for _, hit := range resp.Data["hits"].(map[string]interface{})["hits"].([]interface{}) {
			var destination Destination
			parsedDestination, err := json.Marshal(hit.(map[string]interface{})["_source"].(map[string]interface{})["destination"])
			if err != nil {
				return nil, errors.Wrap(err, "Invalid remote JSON document")
			}
			json.Unmarshal(parsedDestination, &destination)
			destination.ID = hit.(map[string]interface{})["_id"].(string)
			allRemoteDestinationsMap[destination.Name] = destination
		}
	}
	return allRemoteDestinationsMap, nil

}
