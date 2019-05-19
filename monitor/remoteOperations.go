package monitor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"../utils"
	mapset "github.com/deckarep/golang-set"
	"gopkg.in/mihirsoni/yaml.v2"
)

var globalConfig = getConfigYml()

func getConfigYml() Config {
	var globalConfig Config
	yamlFile, err := ioutil.ReadFile("/Users/mihson/openes/alerting-configs/destinations.yml")
	if err != nil {
		fmt.Println("Unable to parse destinations file", err)
		os.Exit(1)
	}
	yaml.Unmarshal(yamlFile, &globalConfig)
	if err != nil {
		fmt.Println("Unable to parse the yml file", err)
		os.Exit(1)
	}
	return globalConfig
}

func GetRemoteMonitors(config ESConfig) (map[string]Monitor, mapset.Set) {
	var (
		allMonitors          []Monitor
		allRemoteMonitorsMap map[string]Monitor
	)
	byt := []byte(`{"query":{ "match_all": {}}}`)
	resp, err := utils.MakeRequest(http.MethodPost,
		config.URL+"_opendistro/_alerting/monitors/_search",
		byt,
		getCommonHeaders(config))
	if err != nil {
		fmt.Println("Error retriving all the monitors", err)
		os.Exit(1)
	}
	// Print the ID and document source for each hit.
	for _, hit := range resp["hits"].(map[string]interface{})["hits"].([]interface{}) {
		var monitor Monitor
		parsedMonitor, err := json.Marshal(hit.(map[string]interface{})["_source"])
		if err != nil {
			fmt.Println("invalid json in the monitor")
			os.Exit(1)
		}
		json.Unmarshal(parsedMonitor, &monitor)
		monitor.id = hit.(map[string]interface{})["_id"].(string)
		flippedDestinations := utils.ReverseMap(globalConfig.Destinations)

		for index := range monitor.Triggers {
			// Update destinationId and actioinId
			for k := range monitor.Triggers[index].Actions {
				destintionName := flippedDestinations[monitor.Triggers[index].Actions[k].DestinationID]
				if destintionName == "" {
					fmt.Println("Looks like remote monitor selected destination doesn't exists here, please update config", monitor.Triggers[index].Actions[k].DestinationID)
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

func PrepareMonitor(localMonitor Monitor, remoteMonitor Monitor, isUpdate bool) Monitor {
	monitorToUpdate := localMonitor
	//Inject triggerIds in case updating existing triggers
	// Convert triggers to map
	remoteTriggers := make(map[string]Trigger)
	if isUpdate == true {
		for _, remoteTrigger := range remoteMonitor.Triggers {
			remoteTriggers[remoteTrigger.Name] = remoteTrigger
		}
	}

	//Update trigger if already existed
	// TODO::Same with Actions once released
	for index := range monitorToUpdate.Triggers {
		// Assume all triggers are new
		monitorToUpdate.Triggers[index].ID = ""
		//Update trigger Id for existing trigger
		if remoteTrigger, ok := remoteTriggers[monitorToUpdate.Triggers[index].Name]; ok && isUpdate {
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

// RunMonitor check if monitor is properly running
func RunMonitor(config ESConfig, id string, monitor Monitor) bool {
	requestBody, err := json.Marshal(monitor)
	if err != nil {
		fmt.Println("Unable to parse monitor Object", err)
		os.Exit(1)
	}
	fmt.Println("requestBody", string(requestBody))
	resp, err := utils.MakeRequest(http.MethodPost,
		config.URL+"_opendistro/_alerting/monitors/_execute?dryrun=true",
		requestBody,
		getCommonHeaders(config))
	if err != nil {
		fmt.Println("Error retriving all the monitors", err)
		os.Exit(1)
	}
	fmt.Println("r", resp)
	res := resp["trigger_results"].(map[string]interface{})
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

func UpdateMonitor(config ESConfig, remoteMonitor Monitor, monitor Monitor) {
	id := remoteMonitor.id
	a, err := json.Marshal(monitor)
	fmt.Printf("%+v\n", monitor)
	if err != nil {
		fmt.Println("Unable to parse monitor Object", err)
		os.Exit(1)
	}
	fmt.Println("Updating existing monitor", string(a))
	resp, err := utils.MakeRequest(http.MethodPut,
		config.URL+"_opendistro/_alerting/monitors/"+id,
		a,
		getCommonHeaders(config))
	if err != nil {
		fmt.Println("Error retriving all the monitors", err)
		os.Exit(1)
	}
	fmt.Println(resp)
}

func CreateNewMonitor(config ESConfig, monitor Monitor) {

	a, err := json.Marshal(monitor)
	fmt.Printf("%+v\n", monitor)
	if err != nil {
		fmt.Println("Unable to parse monitor Object", err)
		os.Exit(1)
	}
	fmt.Println("Updating existing monitor", string(a))
	resp, err := utils.MakeRequest(http.MethodPost,
		config.URL+"_opendistro/_alerting/monitors/",
		a,
		getCommonHeaders(config))
	if err != nil {
		fmt.Println("Error retriving all the monitors", err)
		os.Exit(1)
	}
	fmt.Println(resp)
}
