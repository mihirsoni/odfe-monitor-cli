package monitor

import (
	"github.com/mihirsoni/odfe-alerting/es"
)

func getCommonHeaders(esClient es.Client) map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}
