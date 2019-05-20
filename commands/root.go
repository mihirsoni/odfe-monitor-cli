package commands

import (
	"errors"
	"fmt"
	"os"

	"../es"
	"github.com/spf13/cobra"
)

//Verbose logging if it is true, default to false
var Verbose bool

// ESConfig holds the for ES configuration
var Config es.Config

var esURL string
var userName string
var password string
var rootDir string

// RootCmd asd
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
	cobra.OnInitialize(initEsConfig)
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println(errors.New("Unable to get CWD"))
		os.Exit(1)
	}
	RootCmd.PersistentFlags().StringVarP(&rootDir, "rootDir", "r", dir, "root directory where monitors yml files")
	RootCmd.PersistentFlags().StringVarP(&esURL, "esUrl", "e", "http://localhost:9200/", "URL to connect to Elasticsearch")
	RootCmd.PersistentFlags().StringVarP(&userName, "username", "u", "admin", "URL to connect to Elasticsearch")
	RootCmd.PersistentFlags().StringVarP(&password, "password", "p", "admin", "URL to connect to Elasticsearch")
	RootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
}

func initEsConfig() {
	if esURL != "" && userName != "" && password != "" {
		//Validate URL
		if IsURL(esURL) {
			// Validate ES is running?
			Config = es.Config{URL: esURL, Username: userName, Password: password}
		} else {
			fmt.Println("Invalid URL")
			os.Exit(1)
		}
	} else {
		fmt.Println("Ensure esURL, username and password is set")
	}
}
