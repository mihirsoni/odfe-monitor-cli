package commands

import (
	"github.com/spf13/cobra"
)

var CfgFile string
var Verbose bool

// RootCmd
var RootCmd = &cobra.Command{
	Use:   "This is the simple cli to use",
	Short: "Short description",
	Long: `This application will help you to manage the
            Opendistor Alerting monitors using version controls and configs
            `,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	RootCmd.PersistentFlags().StringVar(&CfgFile, "config", "", "config file (default is $HOME/dagobah/config.yaml)")
	RootCmd.PersistentFlags().String("es_url", "https://localhost:9200", "URL to connect to Elasticsearch")
	RootCmd.PersistentFlags().Bool("versbose", false, "verbose debug")
}
