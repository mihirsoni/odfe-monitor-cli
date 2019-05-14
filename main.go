package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/ghodss/yaml"
	"github.com/mitchellh/mapstructure"
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
	Destination     string `json:"-"`
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
	ID       string    `json:"id,omitempty"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Enabled  bool      `json:"enabled"`
	Schedule Schedule  `json:"schedule"`
	Inputs   []Search  `json:"inputs"`
	Triggers []Trigger `json:"triggers"`
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
func main() {

	localMonitors := getLocalMonitors()
	allRemoteMonitors := getRemoteMonitors()

	localYaml, err := yaml.Marshal(localMonitors["Mihir"])
	remoteYml, err := yaml.Marshal(allRemoteMonitors["Mihir"])
	if err != nil {
		fmt.Printf("Unable to convert into YML")
		os.Exit(1)
	}
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(remoteYml), string(localYaml), true)

	fmt.Println(dmp.DiffPrettyText(diffs))
	runMonitor(allRemoteMonitors["Mihir"].ID, localMonitors["Mihir"])
	updateMonitor(allRemoteMonitors["Mihir"].ID, localMonitors["Mihir"])
	// fmt.Println(len(allRemoteMonitors))
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
func getLocalMonitors() map[string]Monitor {
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
	allLocalMonitorsMap = make(map[string]Monitor)
	for _, localMonitor := range allLocalMonitors {
		for _, trigger := range localMonitor.Triggers {
			for k := range trigger.Actions {
				fmt.Println("Mihir", trigger.Actions[k].Destination)
				//TODO:: Actually read gloabl config
				trigger.Actions[k].DestinationID = "yUC7mWoBPbC8nMZTXQPd"
			}
		}
		allLocalMonitorsMap[localMonitor.Name] = localMonitor
	}

	return allLocalMonitorsMap
}

func getRemoteMonitors() map[string]Monitor {
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
		mapstructure.Decode(hit.(map[string]interface{})["_source"], &monitor)
		monitor.ID = hit.(map[string]interface{})["_id"].(string)
		fmt.Printf("%+v\n", monitor)

		allMonitors = append(allMonitors, monitor)
	}
	allRemoteMonitorsMap = make(map[string]Monitor)
	for _, remoteMonitor := range allMonitors {
		allRemoteMonitorsMap[remoteMonitor.Name] = remoteMonitor
	}
	return allRemoteMonitorsMap
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

func updateMonitor(id string, monitor Monitor) {
	var r map[string]interface{}
	client := http.Client{}
	a, err := json.Marshal(monitor)
	if err != nil {
		fmt.Println("Unable to parse monitor Object")
		os.Exit(1)
	}
	fmt.Println("Id is", string(a))
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
