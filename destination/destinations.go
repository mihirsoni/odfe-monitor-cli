package destination

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mihirsoni/odfe-monitor-cli/es"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

//FileName where destinations are stored and read from
const FileName = "destinations.yaml"
const indexName = ".opendistro-alerting-config"

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
func GetRemote(esClient es.Client) (map[string]string, error) {
	// Adding 10k which will not be the case.
	getAllDestinationQuery := []byte(`{"size": 10000, "query":{ "bool": {"must": { "exists": { "field" : "destination" }}}}}`)
	resp, err := esClient.MakeRequest(http.MethodPost,
		"/.opendistro-alerting-config/_search",
		getAllDestinationQuery,
		getCommonHeaders(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to fetch destinations from elasticsearch")
	}
	var remoteDestinations = make(map[string]string)

	if resp.Status == 200 {
		for _, hit := range resp.Data["hits"].(map[string]interface{})["hits"].([]interface{}) {
			// Improve using gJson , if more complex operation required
			id := hit.(map[string]interface{})["_id"].(string)
			name := hit.(map[string]interface{})["_source"].(map[string]interface{})["destination"].(map[string]interface{})["name"].(string)
			name = strings.ToLower(strings.ReplaceAll(name, " ", "_"))
			remoteDestinations[name] = id
		}
	}
	return remoteDestinations, nil

}
