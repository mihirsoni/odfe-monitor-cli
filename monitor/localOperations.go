package monitor

import (
	"io/ioutil"
	"os"
	"path/filepath"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	"gopkg.in/mihirsoni/yaml.v2"
)

//GetAllLocal Parse all local monitors under rootDir
func GetAllLocal(rootDir string) (map[string]Monitor, mapset.Set, error) {
	var files []string
	var err error
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return nil, nil, errors.Wrap(err, "rootDir does not exist")
	}
	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if info.Name() == "destinations.yml" || info.Name() == "destinations.yaml" || info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".yaml" || ext == ".yml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "Unable to collect all files")
	}
	monitorsSet := mapset.NewSet()
	monitorsMap := make(map[string]Monitor)
	for _, file := range files {
		var allLocalMonitors []Monitor
		yamlFile, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, nil, errors.Wrap(err, "Unable to read "+file)
		}
		err = yaml.Unmarshal(yamlFile, &allLocalMonitors)
		if err != nil {
			return nil, nil, errors.Wrap(err, "Unable to parse "+file)
		}
		for _, localMonitor := range allLocalMonitors {
			if monitorsSet.Contains(localMonitor.Name) {
				return nil, nil, errors.New("Duplicate monitor found. " + localMonitor.Name + " already exists")
			}
			monitorsSet.Add(localMonitor.Name)
			monitorsMap[localMonitor.Name] = localMonitor
		}
	}
	return monitorsMap, monitorsSet, nil
}
