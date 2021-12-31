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

package commands

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/kennygrant/sanitize"
	"github.com/mihirsoni/odfe-monitor-cli/destination"
	"github.com/mihirsoni/odfe-monitor-cli/monitor"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var syncDestinatons bool
var syncMonitors bool
var sync = &cobra.Command{
	Use:   "sync",
	Short: "lets you sync monitors and destinations from remote to local",
	Long:  `This command will fetch all the destinations from ES cluster and write them into a local file in CWD`,
	Args:  validateArgs,
	Run:   runSync,
}

func validateArgs(cmd *cobra.Command, args []string) error {
	if syncDestinatons || syncMonitors {
		return nil
	}
	return errors.New("Provide what to sync monitors or destinations ?  ")
}

func init() {
	sync.Flags().BoolVarP(&syncDestinatons, "destinations", "d", false, "Sync all destinations from ES and write destinations.yml file")
	sync.Flags().BoolVarP(&syncMonitors, "monitors", "m", false, "Sync all monitors from ES and write monitors.yml. Helpful to start from your existing monitors")
	rootCmd.AddCommand(sync)
}

func runSync(cmd *cobra.Command, args []string) {
	destinations, err := destination.GetRemote(esClient)
	check(err)
	if syncDestinatons {
		writeDestinations(destinations)
	} else if syncMonitors {
		monitors, _, err := monitor.GetAllRemote(esClient, destinations)
		check(err)
		writeMonitors(monitors)
	}
}

func writeDestinations(destinations map[string]destination.Destination) {
	destinationsPath := filepath.Join(rootDir, destination.FileName)
	file, err := os.OpenFile(destinationsPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	check(err)
	defer file.Close()
	data, err := yaml.Marshal(destinations)
	check(err)
	file.Write(data)
}

func writeMonitors(monitors map[string]monitor.Monitor) {
	monitorsPath := filepath.Join(rootDir, "monitors/")
	if _, err := os.Stat(monitorsPath); os.IsNotExist(err) {
		os.Mkdir(monitorsPath, 0755)
	}
	for name := range monitors {
		if name == "" {
			log.Info("Monitor with empty name skipped")
			continue
		}
		monitorFile := filepath.Join(monitorsPath, sanitize.BaseName(name)+".yaml")
		file, err := os.OpenFile(monitorFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		check(err)
		data, err := yaml.Marshal(monitors[name])
		check(err)
		file.Write(data)
	}
}
