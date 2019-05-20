package destination

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

func GetLocal(rootDir string) (map[string]string, error) {
	destinations := make(map[string]string)
	destinationsPath := filepath.Join(rootDir, "destinations.yml")
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
	fmt.Println("destinations", destinations)
	return destinations, nil
}
