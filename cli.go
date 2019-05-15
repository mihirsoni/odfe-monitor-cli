package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	mapset "github.com/deckarep/golang-set"
	"github.com/ghodss/yaml"
	flag "github.com/ogier/pflag"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var (
	user string
)

var globalConfig = getConfigYml()

func main() {
	localMonitors, localMonitorSet := getLocalMonitors()
	allRemoteMonitors, remoteMonitorsSet := getRemoteMonitors()
	unTrackedMonitors := remoteMonitorsSet.Difference(localMonitorSet)
	allNewMonitors := localMonitorSet.Difference(remoteMonitorsSet)
	allCommonMonitors := remoteMonitorsSet.Intersect(localMonitorSet)
	fmt.Println("All un tracked monitor", unTrackedMonitors)
	fmt.Println("All new monitor", allNewMonitors)
	fmt.Println("All common monitors", allCommonMonitors)
	changedMonitors := mapset.NewSet()
	allCommonMonitorsIt := allCommonMonitors.Iterator()
	for commonMonitor := range allCommonMonitorsIt.C {
		if isMonitorChanged(localMonitors[commonMonitor.(string)], allRemoteMonitors[commonMonitor.(string)]) != true {
			changedMonitors.Add(commonMonitor)
		}
	}
	fmt.Println("monitors to be updated", changedMonitors)
	for monitorToBeUpdated := range changedMonitors.Iterator().C {
		monitorName := monitorToBeUpdated.(string)
		localYaml, err := yaml.Marshal(localMonitors[monitorName])
		remoteYml, err := yaml.Marshal(allRemoteMonitors[monitorName])
		if err != nil {
			fmt.Printf("Unable to convert into YML")
			os.Exit(1)
		}
		fmt.Println("remoteYml", string(remoteYml))
		dmp := diffmatchpatch.New()

		diffs := dmp.DiffMain(string(remoteYml), string(localYaml), false)

		fmt.Println(dmp.DiffPrettyText(diffs))
		diff := cmp.Diff(allRemoteMonitors[monitorName], localMonitors[monitorName], cmpopts.IgnoreUnexported(Monitor{}))
		fmt.Println(string(diff))
		canonicalMonitor := prepareMonitor(localMonitors[monitorName], allRemoteMonitors[monitorName])
		runMonitor(allRemoteMonitors[monitorName].id, canonicalMonitor)
		updateMonitor(allRemoteMonitors[monitorName], canonicalMonitor)
	}
	// fmt.Println(len(allRemoteMonitors))
}
func isMonitorChanged(localMonitor Monitor, remoteMonitor Monitor) bool {
	return cmp.Equal(localMonitor, remoteMonitor, cmpopts.IgnoreUnexported(Monitor{}))
}

func getConfigYml() Config {
	var globalConfig Config
	yamlFile, err := ioutil.ReadFile("config.yml")
	if err != nil {
		fmt.Println("Unable to parse monitor file", err)
		os.Exit(1)
	}
	yaml.Unmarshal(yamlFile, &globalConfig)
	if err != nil {
		fmt.Println("Unable to parse the yml file", err)
		os.Exit(1)
	}
	return globalConfig
}

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

func init() {
	flag.StringVarP(&user, "user", "u", "", "Search Users")
}

func getLocalMonitors() (map[string]Monitor, mapset.Set) {
	var allLocalMonitorsMap map[string]Monitor
	var allLocalMonitors []Monitor
	yamlFile, err := ioutil.ReadFile("monitor.yml")
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
