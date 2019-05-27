package commands

import (
	"net/http"
	"os"
	"strings"

	"../es"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//Verbose logging if it is true, default to false
var Verbose bool

// ESConfig holds the for ES configuration
var esClient es.Client

var esURL string
var userName string
var password string
var rootDir string

// RootCmd asd
var rootCmd = &cobra.Command{
	Use:   "od-alerting-cli [COMMAND] [OPTIONS]",
	Short: "One stop solution for managing your monitors",
	Long: `This application will help you to manage the
            Opendistro Alerting monitors using yaml files
            `,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	cobra.OnInitialize(setup)
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("Unable to get CWD", err)
	}
	rootCmd.PersistentFlags().StringVarP(&rootDir, "rootDir", "r", dir, "root directory where monitors yml files")
	rootCmd.PersistentFlags().StringVarP(&esURL, "esUrl", "e", "https://localhost:9200/", "URL to connect to Elasticsearch")
	rootCmd.PersistentFlags().StringVarP(&userName, "username", "u", "admin", "URL to connect to Elasticsearch")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "admin", "URL to connect to Elasticsearch")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
}

func setup() {
	if esURL != "" {
		//Validate URL
		if IsURL(esURL) {
			// Validate ES is running?
			trailing := strings.HasSuffix(esURL, "/")
			if trailing {
				esURL = strings.TrimSuffix(esURL, "/")
			}
			esClient = es.Client{URL: esURL, Username: userName, Password: password}
			resp, err := esClient.MakeRequest(http.MethodGet, "", nil, nil)
			check(err)
			if resp.Status != 200 {
				log.Fatal("Unable to connect to elasticsearch")
			}

		} else {
			log.WithFields(log.Fields{"elasticsearch-url": esURL}).Fatal("Elasticsearch url is invalid")
		}
	} else {
		// Solve with required flags
		log.Fatal("Ensure esURL is provided")
	}

	if Verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
