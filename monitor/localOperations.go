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

package monitor

import (
	"io/ioutil"
	"os"
	"path/filepath"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
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
