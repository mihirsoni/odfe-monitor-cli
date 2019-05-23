package monitor

import (
	"encoding/json"
	"net/http"
	"strconv"

	"../es"
	"../utils"
	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
)

// GetAllRemote will pull all the monitors from ES cluster
func GetAllRemote(config es.Config, destinationsMap map[string]string) (map[string]Monitor, mapset.Set, error) {
	var (
		allMonitors          []Monitor
		allRemoteMonitorsMap map[string]Monitor
	)
	byt := []byte(`{"query":{ "match_all": {}}}`)
	resp, err := es.MakeRequest(http.MethodPost,
		config.URL+"_opendistro/_alerting/monitors/_search",
		byt,
		getCommonHeaders(config))
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error retriving all the monitors")
	}
	// Print the ID and document source for each hit.
	for _, hit := range resp.Data["hits"].(map[string]interface{})["hits"].([]interface{}) {
		var monitor Monitor
		parsedMonitor, err := json.Marshal(hit.(map[string]interface{})["_source"])
		if err != nil {
			return nil, nil, errors.Wrap(err, "Invalid remote JSON document")
		}
		json.Unmarshal(parsedMonitor, &monitor)
		monitor.id = hit.(map[string]interface{})["_id"].(string)
		monitor.primaryTerm = strconv.FormatFloat(hit.(map[string]interface{})["_primary_term"].(float64), 'f', 0, 64)
		monitor.seqNo = strconv.FormatFloat(hit.(map[string]interface{})["_seq_no"].(float64), 'f', 0, 64)
		flippedDestinations := utils.ReverseMap(destinationsMap)

		for index := range monitor.Triggers {
			// flip DestinationsId
			for k := range monitor.Triggers[index].Actions {
				destintionName := flippedDestinations[monitor.Triggers[index].Actions[k].DestinationID]
				if destintionName == "" {
					return nil, nil, errors.New("Remote monitor selected destination doesn't exists locally, please update destinations list if out of sync")
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
	return allRemoteMonitorsMap, remoteMonitorsSet, nil
}

// Prepare will modify the monitor to populate correct IDs
func Prepare(localMonitor Monitor,
	remoteMonitor Monitor,
	destinationsMap map[string]string,
	isUpdate bool) (Monitor, error) {
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
			destinationName := monitorToUpdate.Triggers[index].Actions[k].DestinationID
			destinationID := destinationsMap[destinationName]
			if destinationID == "" {
				return monitorToUpdate,
					errors.New("Specified destination " + destinationName +
						" in monitor " + monitorToUpdate.Name +
						" doesn't exist in destinations list, sync destinations using sync --destination")
			}
			monitorToUpdate.Triggers[index].Actions[k].DestinationID = destinationID
			//Update action Id for existing action instead of creating new
			if remoteAction, ok := remoteActions[monitorToUpdate.Triggers[index].Actions[k].Name]; ok && isUpdate {
				monitorToUpdate.Triggers[index].Actions[k].ID = remoteAction.ID
			}
		}
	}
	return monitorToUpdate, nil
}

// Run will execute monitor
func Run(config es.Config, monitor Monitor, ch chan<- error) {
	requestBody, err := json.Marshal(monitor)
	if err != nil {
		ch <- errors.Wrap(err, "Unable to parse monitor correctly")
	}
	resp, err := es.MakeRequest(http.MethodPost,
		config.URL+"_opendistro/_alerting/monitors/_execute?dryrun=true",
		requestBody,
		getCommonHeaders(config))
	if err != nil {
		ch <- errors.Wrap(err, "Unable to execute monitor")
	}

	monitorError, _ := resp.Data["error"].(map[string]interface{})
	if monitorError != nil {
		indentJSON, _ := json.MarshalIndent(monitorError, "", "\t")
		ch <- errors.New("Error executing monitor " + monitor.Name + "\n" + string(indentJSON))
		return
	}
	executionResult, err := json.Marshal(resp.Data["trigger_results"].(map[string]interface{}))
	if err != nil {
		ch <- errors.Wrap(err, "Unable to parse run monitor response")
	}
	var triggersResult interface{}
	json.Unmarshal(executionResult, &triggersResult)
	triggersResultMap := triggersResult.(map[string]interface{})
	for _, result := range triggersResultMap {
		// Convert response and validate if any error
		var runResult map[string]interface{}
		parsedResultSet, err := json.Marshal(result)
		if err != nil {
			ch <- errors.Wrap(err, "Unable to parse trigger result correctly")
		}
		json.Unmarshal(parsedResultSet, &runResult)
		if runResult["error"] != nil {
			indentJSON, _ := json.MarshalIndent(runResult, "", "\t")
			ch <- errors.New(string(indentJSON))
		}
	}
	ch <- nil
}

// Update will modify existing monitor
func Update(config es.Config, remoteMonitor Monitor, monitor Monitor, ch chan<- error) {
	requestBody, err := json.Marshal(monitor)
	if err != nil {
		ch <- errors.Wrap(err, "Unable to parse monitor Object "+monitor.Name)
	}
	resp, err := es.MakeRequest(http.MethodPut,
		config.URL+"_opendistro/_alerting/monitors/"+remoteMonitor.id+
			"?if_seq_no="+remoteMonitor.seqNo+
			"&if_primary_term="+remoteMonitor.primaryTerm,
		requestBody,
		getCommonHeaders(config))
	if err != nil {
		ch <- errors.Wrap(err, "Unable to update monitor "+monitor.Name)
	}
	if resp.Status != 200 {
		indentJSON, _ := json.MarshalIndent(resp.Data, "", "\t")
		ch <- errors.New("Unable to update monitor" + monitor.Name + " " + string(indentJSON))
	}
	ch <- nil
}

// Create will create new monitor
func Create(config es.Config, monitor Monitor, ch chan<- error) {
	requestBody, err := json.Marshal(monitor)
	if err != nil {
		ch <- errors.Wrap(err, "Unable to parse monitor Object "+monitor.Name)
	}
	resp, err := es.MakeRequest(http.MethodPost,
		config.URL+"_opendistro/_alerting/monitors/",
		requestBody,
		getCommonHeaders(config))
	if err != nil {
		ch <- errors.Wrap(err, "Unable to create new Monitor")
	}
	if resp.Status != 201 {
		indentJSON, _ := json.MarshalIndent(resp.Data, "", "\t")
		ch <- errors.New("Unable to create monitor " + monitor.Name + string(indentJSON))
	}
	ch <- nil
}

// Delete delete un-tracked monitor
func Delete(config es.Config, monitor Monitor, ch chan<- error) {
	var requestBody []byte
	resp, err := es.MakeRequest(http.MethodDelete,
		config.URL+"_opendistro/_alerting/monitors/"+monitor.id,
		requestBody,
		getCommonHeaders(config))
	if err != nil {
		ch <- errors.Wrap(err, "Unable to delete a monitor "+monitor.Name)
	}
	if resp.Status != 200 {
		ch <- errors.New("Unable to delete monitor" + monitor.Name + " ")
	}
	ch <- nil
}
