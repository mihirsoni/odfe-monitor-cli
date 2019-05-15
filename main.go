package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

// Search hello
type Search struct {
	Search struct {
		Indices []string               `json:"indices"`
		Query   map[string]interface{} `json:"query"`
	} `json:"search"`
}

type Trigger struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Severity  string    `json:"severity"`
	Condition Condition `json:"condition"`
	Actions   []Action  `json:"actions,omitempty"`
}

// Period hello
type Period struct {
	Interval int    `json:"interval"`
	Unit     string `json:"unit"`
}

// Cron hello
type Cron struct {
	Expression string `json:"expression"`
	Timezone   string `json:"timezone"`
}

// Schedule world
type Schedule struct {
	Period *Period `json:"period,omitempty"`
	Cron   *Cron   `json:"cron,omitempty"`
}

//Action action model
type Action struct {
	Name            string `json:"name"`
	DestinationID   string `json:"destination_id,omitempty"`
	SubjectTemplate struct {
		Source string `json:"source"`
		Lang   string `json:"lang"`
	} `json:"subject_template"`
	MessageTemplate struct {
		Source string `json:"source"`
		Lang   string `json:"lang"`
	} `json:"message_template"`
}

type Script struct {
	Source string `json:"source"`
	Lang   string `json:"lang"`
}
type Condition struct {
	Script Script `json:"script"`
}

// Monitor nice
type Monitor struct {
	id       string
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Enabled  bool      `json:"enabled"`
	Schedule Schedule  `json:"schedule"`
	Inputs   []Search  `json:"inputs"`
	Triggers []Trigger `json:"triggers"`
}

type Config struct {
	Destinations map[string]string
}

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
func (monitor *Monitor) getMonitor() *Monitor {
	yamlFile, err := ioutil.ReadFile("monitor.yml")
	if err != nil {
		fmt.Println("Unable to parse monitor file", err)
		os.Exit(1)
	}
	err = yaml.Unmarshal(yamlFile, &monitor)
	if err != nil {
		fmt.Println("Unable to parse the yml file asda", err)
		os.Exit(1)
	}
	return monitor
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

func diff() {
	//All New
	// Modified
	//
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

func reverseMap(m map[string]string) map[string]string {
	n := make(map[string]string)
	for k, v := range m {
		n[v] = k
	}
	return n
}

func getRemoteMonitors() (map[string]Monitor, mapset.Set) {
	var (
		r                    map[string]interface{}
		allMonitors          []Monitor
		allRemoteMonitorsMap map[string]Monitor
	)
	byt := []byte(`{"query":{ "match_all": {}}}`)
	resp, err := http.Post("http://localhost:9200/_opendistro/_alerting/monitors/_search", "application/json", bytes.NewBuffer(byt))
	if err != nil {
		fmt.Println("Error retriving all the monitors", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&r)
	// Print the ID and document source for each hit.
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		var monitor Monitor
		parsedMonitor, err := json.Marshal(hit.(map[string]interface{})["_source"])
		if err != nil {
			fmt.Println("invalid json in the monitor")
			os.Exit(1)
		}
		json.Unmarshal(parsedMonitor, &monitor)
		monitor.id = hit.(map[string]interface{})["_id"].(string)
		flippedDestinations := reverseMap(globalConfig.Destinations)

		for index := range monitor.Triggers {
			// Update destinationId and actioinId
			for k := range monitor.Triggers[index].Actions {
				destintionName := flippedDestinations[monitor.Triggers[index].Actions[k].DestinationID]
				if destintionName == "" {
					fmt.Println("Looks like remote monitor selected destination doesn't exists here, please update config")
					os.Exit(1)
				}
				monitor.Triggers[index].Actions[k].DestinationID = destintionName
			}
		}
		allMonitors = append(allMonitors, monitor)
	}
	allRemoteMonitorsMap = make(map[string]Monitor)
	remoteMonitorsSet := mapset.NewSet()
	for _, remoteMonitor := range allMonitors {
		remoteMonitorsSet.Add(remoteMonitor.Name)
		allRemoteMonitorsMap[remoteMonitor.Name] = remoteMonitor
	}
	return allRemoteMonitorsMap, remoteMonitorsSet
}

func prepareMonitor(localMonitor Monitor, remoteMonitor Monitor) Monitor {
	monitorToUpdate := localMonitor
	//Inject triggerIds in case updating existing triggers
	// Convert triggers to map
	remoteTriggers := make(map[string]Trigger)
	for _, remoteTrigger := range remoteMonitor.Triggers {
		remoteTriggers[remoteTrigger.Name] = remoteTrigger
	}
	//Update trigger if already existed
	// TODO::Same with Actions once released
	for index := range monitorToUpdate.Triggers {
		//Update trigger Id
		if remoteTrigger, ok := remoteTriggers[monitorToUpdate.Triggers[index].Name]; ok {
			monitorToUpdate.Triggers[index].ID = remoteTrigger.ID
		}
		// Update destinationId and actioinId
		for k := range monitorToUpdate.Triggers[index].Actions {
			destinationID := globalConfig.Destinations[monitorToUpdate.Triggers[index].Actions[k].DestinationID]
			if destinationID == "" {
				fmt.Println("destination specified doesn't exist in config file, verify it")
				os.Exit(1)
			}
			monitorToUpdate.Triggers[index].Actions[k].DestinationID = destinationID
		}
	}
	return monitorToUpdate
}

// TODO , check if the query is incorrect
func runMonitor(id string, monitor Monitor) bool {
	var r map[string]interface{}
	requestBody, err := json.Marshal(monitor)
	fmt.Println("requestBody", string(requestBody))
	resp, err := http.Post("http://localhost:9200/_opendistro/_alerting/monitors/_execute?dryrun=true", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Error retriving all the monitors", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&r)
	fmt.Println("r", r)
	res := r["trigger_results"].(map[string]interface{})
	executionResult, err := json.Marshal(res)
	var t interface{}
	err = json.Unmarshal(executionResult, &t)
	itemsMap := t.(map[string]interface{})
	for _, v := range itemsMap {
		var val map[string]interface{}
		asd, err := json.Marshal(v)
		if err != nil {
			fmt.Println("unable to find the proper response ")
			os.Exit(1)
		}
		json.Unmarshal(asd, &val)
		if val["error"] != nil {
			fmt.Println("Unable to run the monitor", val["error"])
			os.Exit(1)
		}
	}
	return true
}

func updateMonitor(remoteMonitor Monitor, monitor Monitor) {
	id := remoteMonitor.id
	var r map[string]interface{}
	client := http.Client{}
	a, err := json.Marshal(monitor)
	if err != nil {
		fmt.Println("Unable to parse monitor Object")
		os.Exit(1)
	}
	fmt.Println("Updating existing monitor", string(a))
	req, err := http.NewRequest(http.MethodPut, "http://localhost:9200/_opendistro/_alerting/monitors/"+id, bytes.NewBuffer(a))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error retriving all the monitors", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&r)
	fmt.Println(r)
}
