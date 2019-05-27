package commands

import (
	"os"
	"path/filepath"

	"../destination"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var syncDestinatons bool
var sync = &cobra.Command{
	Use:   "sync [Flags]",
	Short: "sync operation",
	Long:  `This command will fetch all the destinations from ES cluster and write them into a local file`,
	Run:   runSync,
}

func runSync(cmd *cobra.Command, args []string) {
	if syncDestinatons {
		destinations, err := destination.GetRemote(Config)
		check(err)
		writeDestinations(destinations)
	}
}

func writeDestinations(destinations map[string]string) {
	destinationsPath := filepath.Join(rootDir, destination.FILE_NAME)
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

func init() {
	sync.Flags().BoolVarP(&syncDestinatons, "destinations", "d", false, "sync all destinations from ES and write destinations.yml file")
	rootCmd.AddCommand(sync)
}
