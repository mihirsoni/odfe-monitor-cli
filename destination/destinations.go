package destination

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mihirsoni/od-alerting-cli/es"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

//FILE_NAME where destiantions are stored and read from
const FILE_NAME = "destinations.yaml"

func GetLocal(rootDir string) (map[string]string, error) {
	destinations := make(map[string]string)
	destinationsPath := filepath.Join(rootDir, FILE_NAME)
	if _, err := os.Stat(destinationsPath); os.IsNotExist(err) {
		return nil, errors.Wrap(err, "destinations.yml doesn't exist")
	}
	yamlFile, err := ioutil.ReadFile(destinationsPath)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to read destinations file")
	}
	yaml.Unmarshal(yamlFile, &destinations)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to parse destinations file , invalid yml ?")
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
		"/_search",
		getAllDestinationQuery,
		getCommonHeaders(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to fetch destinations")
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
