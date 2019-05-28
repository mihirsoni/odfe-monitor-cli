package monitor

import (
	"github.com/mihirsoni/od-alerting-cli/es"
)

func getCommonHeaders(esClient es.Client) map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}
