package monitor

import (
	"io/ioutil"
	"os"
	"path/filepath"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	"gopkg.in/mihirsoni/yaml.v2"
)

func GetAllLocal(rootDir string) (map[string]Monitor, mapset.Set, error) {
	var monitorsMap map[string]Monitor
	var allLocalMonitors []Monitor
	var files []string
	var err error
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return nil, nil, errors.Wrap(err, "rootDir does not exist")
	}
	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if info.Name() == "destinations.yml" || info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "Unable to collect all files")
	}
	monitorsSet := mapset.NewSet()
	monitorsMap = make(map[string]Monitor)
	for _, file := range files {
		var yamlFile []byte
		yamlFile, err = ioutil.ReadFile(file)
		if err != nil {
			return nil, nil, errors.Wrap(err, "Unable to read monitor file "+file)
		}
		err = yaml.Unmarshal(yamlFile, &allLocalMonitors)
		if err != nil {
			return nil, nil, errors.Wrap(err, "Unable to parse monitor "+file)
		}
		for _, localMonitor := range allLocalMonitors {
			if monitorsSet.Contains(localMonitor.Name) {
				return nil, nil, errors.New("Duplicate monitor found " + localMonitor.Name + " already exists")
			}
			monitorsSet.Add(localMonitor.Name)
			monitorsMap[localMonitor.Name] = localMonitor
		}
	}
	return monitorsMap, monitorsSet, nil
}
