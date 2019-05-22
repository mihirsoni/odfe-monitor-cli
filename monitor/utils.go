package monitor

import (
	"../es"
)

func getCommonHeaders(esConfig es.Config) map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}
