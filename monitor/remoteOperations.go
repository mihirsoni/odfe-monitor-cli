package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

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
		fmt.Println(strconv.FormatFloat(hit.(map[string]interface{})["_primary_term"].(float64), 'f', 0, 64))
		monitor.primaryTerm = strconv.FormatFloat(hit.(map[string]interface{})["_primary_term"].(float64), 'f', 0, 64)
		monitor.seqNo = strconv.FormatFloat(hit.(map[string]interface{})["_seq_no"].(float64), 'f', 0, 64)
		flippedDestinations := utils.ReverseMap(globalConfig.Destinations)

		for index := range monitor.Triggers {
			// flip DestinationsId
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
		remoteActions := make(map[string]Action)
		if isUpdate == true {
			for _, remoteAction := range remoteTriggers[monitorToUpdate.Triggers[index].Name].Actions {
				remoteActions[remoteAction.Name] = remoteAction
			}
		}
		for k := range monitorToUpdate.Triggers[index].Actions {
			monitorToUpdate.Triggers[index].Actions[k].ID = ""
			destinationID := globalConfig.Destinations[monitorToUpdate.Triggers[index].Actions[k].DestinationID]
			if destinationID == "" {
				fmt.Println("destination specified doesn't exist in config file, verify it")
				os.Exit(1)
			}
			monitorToUpdate.Triggers[index].Actions[k].DestinationID = destinationID
			//Update action Id for existing action instead of creating new
			if remoteAction, ok := remoteActions[monitorToUpdate.Triggers[index].Actions[k].Name]; ok && isUpdate {
				monitorToUpdate.Triggers[index].Actions[k].ID = remoteAction.ID
			}
		}
	}
	return monitorToUpdate
}

// RunMonitor check if monitor is properly running
func RunMonitor(config ESConfig, monitor Monitor) (bool, error) {
	requestBody, err := json.Marshal(monitor)
	if err != nil {
		return false, err
	}
	resp, err := utils.MakeRequest(http.MethodPost,
		config.URL+"_opendistro/_alerting/monitors/_execute?dryrun=true",
		requestBody,
		getCommonHeaders(config))
	if err != nil {
		return false, errors.New("Unable to execute monitor")
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
			return false, err
		}
		json.Unmarshal(asd, &val)
		if val["error"] != nil {
			return false, errors.New(val["error"].(string))
		}
	}
	return false, nil
}

func UpdateMonitor(config ESConfig, remoteMonitor Monitor, monitor Monitor) {
	id := remoteMonitor.id
	a, err := json.Marshal(monitor)
	fmt.Printf("%+v\n", monitor)
	if err != nil {
		fmt.Println("Unable to parse monitor Object", err)
		os.Exit(1)
	}
	resp, err := utils.MakeRequest(http.MethodPut,
		config.URL+"_opendistro/_alerting/monitors/"+id+
			"?if_seq_no="+remoteMonitor.seqNo+
			"&if_primary_term="+remoteMonitor.primaryTerm,
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
