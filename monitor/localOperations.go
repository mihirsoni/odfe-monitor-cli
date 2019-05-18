package monitor

import (
	"fmt"
	"io/ioutil"
	"os"

	mapset "github.com/deckarep/golang-set"
	"gopkg.in/mihirsoni/yaml.v2"
)

func GetLocalMonitors() (map[string]Monitor, mapset.Set) {
	var allLocalMonitorsMap map[string]Monitor
	var allLocalMonitors []Monitor
	yamlFile, err := ioutil.ReadFile("/Users/mihson/openes/alerting-configs/monitor.yml")
	if err != nil {
		fmt.Println("Unable to parse monitor file", err)
		os.Exit(1)
	}
	yaml.Unmarshal(yamlFile, &allLocalMonitors)
	if err != nil {
		fmt.Println("Unable to parse the yml file", err)
		os.Exit(1)
	}
	//Validate uniq name
	checkUniqueMonitorNames(allLocalMonitors)
	localMonitorSet := mapset.NewSet()
	allLocalMonitorsMap = make(map[string]Monitor)
	for _, localMonitor := range allLocalMonitors {
		localMonitorSet.Add(localMonitor.Name)
		allLocalMonitorsMap[localMonitor.Name] = localMonitor
	}

	return allLocalMonitorsMap, localMonitorSet
}
