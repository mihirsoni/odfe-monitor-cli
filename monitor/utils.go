package monitor

import (
	"../es"
)

func getCommonHeaders(esClient es.Client) map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}
