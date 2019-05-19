package monitor

import (
	"fmt"
	"os"
)

func checkUniqueMonitorNames(monitors []Monitor) bool {
	count := make(map[string]int)
	for _, monitor := range monitors {
		if count[monitor.Name] > 0 {
			fmt.Println("Duplicate name exists all monitor name should be unique")
			os.Exit(1)
		}
		count[monitor.Name] = 1
	}
	return true
}

func getCommonHeaders(esConfig ESConfig) map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}
