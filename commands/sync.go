package commands

import (
	"os"
	"path/filepath"

	"github.com/mihirsoni/odfe-alerting/destination"
	"github.com/mihirsoni/odfe-alerting/monitor"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var syncDestinatons bool
var syncMonitors bool
var sync = &cobra.Command{
	Use:   "sync",
	Short: "lets you sync monitors and destinations from remote to local",
	Long:  `This command will fetch all the destinations from ES cluster and write them into a local file in CWD`,
	Run:   runSync,
}

func init() {
	sync.Flags().BoolVarP(&syncDestinatons, "destinations", "d", false, "sync all destinations from ES and write destinations.yml file")
	sync.Flags().BoolVarP(&syncMonitors, "monitors", "m", false, "sync all monitors from ES and write monitors.yml. Helpful to start from your existing monitors")
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

func writeDestinations(destinations map[string]string) {
	destinationsPath := filepath.Join(rootDir, destination.FileName)
	if _, err := os.Stat(destinationsPath); os.IsNotExist(err) {
		_, err = os.Create(destinationsPath)
		check(err)
	}
	file, err := os.OpenFile(destinationsPath, os.O_WRONLY, 0644)
	check(err)
	defer file.Close()
	data, err := yaml.Marshal(destinations)
	check(err)
	file.Write(data)
}

func writeMonitors(monitors map[string]monitor.Monitor) {
	destinationsPath := filepath.Join(rootDir, "monitors.yaml")
	if _, err := os.Stat(destinationsPath); os.IsNotExist(err) {
		_, err = os.Create(destinationsPath)
		check(err)
	}
	file, err := os.OpenFile(destinationsPath, os.O_WRONLY, 0644)
	check(err)
	defer file.Close()
	var monitorsList []monitor.Monitor
	for name := range monitors {
		monitorsList = append(monitorsList, monitors[name])
	}
	data, err := yaml.Marshal(monitorsList)
	check(err)
	file.Write(data)
}
