package monitor

import (
	"encoding/json"
	"net/http"
	"strconv"

	mapset "github.com/deckarep/golang-set"
	"github.com/mihirsoni/odfe-alerting/es"
	"github.com/mihirsoni/odfe-alerting/utils"
	"github.com/pkg/errors"
)

// GetAllRemote will pull all the monitors from ES cluster
func GetAllRemote(esClient es.Client, destinationsMap map[string]string) (map[string]Monitor, mapset.Set, error) {
	var (
		allMonitors          []Monitor
		allRemoteMonitorsMap map[string]Monitor
	)
	// Since this is very simple call to match all maximum monitors which is 1000 for now
	byt := []byte(`{"size": 1000, "query":{ "match_all": {}}}`)
	resp, err := esClient.MakeRequest(http.MethodPost,
		"/_opendistro/_alerting/monitors/_search",
		byt,
		getCommonHeaders(esClient))
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error retriving all the monitors")
	}
	allRemoteMonitorsMap = make(map[string]Monitor)
	remoteMonitorsSet := mapset.NewSet()
	if resp.Status == 404 {
		// No monitors exists so no index exists, returning empty and will create new monitors
		return allRemoteMonitorsMap, remoteMonitorsSet, nil
	}
	for _, hit := range resp.Data["hits"].(map[string]interface{})["hits"].([]interface{}) {
		var monitor Monitor
		parsedMonitor, err := json.Marshal(hit.(map[string]interface{})["_source"])
		if err != nil {
			return nil, nil, errors.Wrap(err, "Invalid remote JSON document")
		}
		json.Unmarshal(parsedMonitor, &monitor)
		monitor.id = hit.(map[string]interface{})["_id"].(string)
		// Old version doesn't have primary term or seq_no
		if esClient.Version > 0 {
			monitor.primaryTerm = strconv.FormatFloat(hit.(map[string]interface{})["_primary_term"].(float64), 'f', 0, 64)
			monitor.seqNo = strconv.FormatFloat(hit.(map[string]interface{})["_seq_no"].(float64), 'f', 0, 64)
		}
		flippedDestinations := utils.ReverseMap(destinationsMap)

		for index := range monitor.Triggers {
			//Modify the condition
			monitor.Triggers[index].YCondition = monitor.Triggers[index].Condition.Script.Source
			// flip DestinationsId
			for k := range monitor.Triggers[index].Actions {
				destinationID := monitor.Triggers[index].Actions[k].DestinationID
				destintionName := flippedDestinations[destinationID]
				if destintionName == "" {
					return nil, nil, errors.New("Invalid destination" + destinationID + " in monitor " +
						monitor.Name + ".If out of sync update using --sync --destination or update")
				}
				monitor.Triggers[index].Actions[k].Subject = monitor.Triggers[index].Actions[k].SubjectTemplate.Source
				monitor.Triggers[index].Actions[k].Message = monitor.Triggers[index].Actions[k].MessageTemplate.Source
				monitor.Triggers[index].Actions[k].DestinationID = destintionName
			}
		}
		allMonitors = append(allMonitors, monitor)
	}
	for _, remoteMonitor := range allMonitors {
		remoteMonitorsSet.Add(remoteMonitor.Name)
		allRemoteMonitorsMap[remoteMonitor.Name] = remoteMonitor
	}
	return allRemoteMonitorsMap, remoteMonitorsSet, nil
}

// Prepare will modify the monitor to populate correct IDs
func (monitor *Monitor) Prepare(
	remoteMonitor Monitor,
	destinationsMap map[string]string,
	isUpdate bool) error {

	monitor.seqNo = remoteMonitor.seqNo
	monitor.primaryTerm = remoteMonitor.primaryTerm
	monitor.id = remoteMonitor.id
	//Inject triggerIds in case updating existing triggers
	// Convert triggers to map
	remoteTriggers := make(map[string]Trigger)
	if isUpdate == true {
		for _, remoteTrigger := range remoteMonitor.Triggers {
			remoteTriggers[remoteTrigger.Name] = remoteTrigger
		}
	}
	//Update trigger if already existed
	//TODO:: break down this operation
	for index := range monitor.Triggers {
		// Assume all triggers are new
		monitor.Triggers[index].ID = ""
		//Update trigger Id for existing trigger
		if remoteTrigger, ok := remoteTriggers[monitor.Triggers[index].Name]; ok && isUpdate {
			monitor.Triggers[index].ID = remoteTrigger.ID
		}
		//Simplify condition
		monitor.Triggers[index].Condition = Condition{
			Script{
				Source: monitor.Triggers[index].YCondition,
				Lang:   "painless",
			},
		}
		// Update destinationId and actionID
		remoteActions := make(map[string]Action)
		if isUpdate == true {
			for _, remoteAction := range remoteTriggers[monitor.Triggers[index].Name].Actions {
				remoteActions[remoteAction.Name] = remoteAction
			}
		}
		for k := range monitor.Triggers[index].Actions {
			currentAction := monitor.Triggers[index].Actions[k]
			currentAction.ID = ""
			remoteDestinationID := destinationsMap[currentAction.DestinationID]
			if remoteDestinationID == "" {
				return errors.New("Specified destination " + currentAction.DestinationID +
					" in monitor " + monitor.Name +
					" doesn't exist in destinations list, sync destinations using sync --destination")
			}
			currentAction.DestinationID = remoteDestinationID
			// Converting subject to adhere to API
			currentAction.SubjectTemplate = Script{
				Source: currentAction.Subject,
				Lang:   "mustache",
			}
			currentAction.MessageTemplate = Script{
				Source: currentAction.Message,
				Lang:   "mustache",
			}
			//Update action Id for existing action instead of creating new
			if remoteAction, ok := remoteActions[currentAction.Name]; ok && isUpdate {
				currentAction.ID = remoteAction.ID
			}
			monitor.Triggers[index].Actions[k] = currentAction
		}
	}
	return nil
}

// Run will execute monitor
func (monitor *Monitor) Run(esClient es.Client, dryRun bool) error {
	requestBody, err := json.Marshal(monitor)
	// fmt.Println("monitor", string(requestBody))
	if err != nil {
		return errors.Wrap(err, "Unable to parse monitor correctly")
	}
	resp, err := esClient.MakeRequest(http.MethodPost,
		"/_opendistro/_alerting/monitors/_execute?dryrun="+strconv.FormatBool(dryRun),
		requestBody,
		getCommonHeaders(esClient))
	if err != nil {
		return errors.Wrap(err, "Unable to execute monitor")
	}

	monitorError, _ := resp.Data["error"].(map[string]interface{})
	if monitorError != nil {
		indentJSON, _ := json.MarshalIndent(monitorError, "", "\t")
		return errors.New("Error executing monitor " + monitor.Name + "\n" + string(indentJSON))

	}
	executionResult, err := json.Marshal(resp.Data["trigger_results"].(map[string]interface{}))
	if err != nil {
		return errors.Wrap(err, "Unable to parse run monitor response")
	}
	var triggersResult interface{}
	json.Unmarshal(executionResult, &triggersResult)
	triggersResultMap := triggersResult.(map[string]interface{})
	for _, result := range triggersResultMap {
		// Convert response and validate if any error
		var runResult map[string]interface{}
		parsedResultSet, err := json.Marshal(result)
		if err != nil {
			return errors.Wrap(err, "Unable to parse trigger result correctly")
		}
		json.Unmarshal(parsedResultSet, &runResult)
		if runResult["error"] != nil {
			indentJSON, _ := json.MarshalIndent(runResult, "", "\t")
			return errors.New(string(indentJSON))
		}
	}
	return nil
}

// Update will modify existing monitor
func (monitor *Monitor) Update(esClient es.Client) error {
	requestBody, err := json.Marshal(monitor)
	if err != nil {
		return errors.Wrap(err, "Unable to parse monitor Object "+monitor.Name)
	}
	endPoint := "/_opendistro/_alerting/monitors/" + monitor.id
	if esClient.Version > 0 {
		endPoint = endPoint + "?if_seq_no=" + monitor.seqNo + "&if_primary_term=" + monitor.primaryTerm
	}
	resp, err := esClient.MakeRequest(http.MethodPut,
		endPoint,
		requestBody,
		getCommonHeaders(esClient))
	if err != nil {
		return errors.Wrap(err, "Unable to update monitor "+monitor.Name)
	}
	if resp.Status != 200 {
		indentJSON, _ := json.MarshalIndent(resp.Data, "", "\t")
		return errors.New("Unable to update monitor" + monitor.Name + " " + string(indentJSON))
	}
	return nil
}

// Create will create new monitor
func (monitor *Monitor) Create(esClient es.Client) error {
	requestBody, err := json.Marshal(monitor)
	if err != nil {
		return errors.Wrap(err, "Unable to parse monitor Object "+monitor.Name)
	}
	resp, err := esClient.MakeRequest(http.MethodPost,
		"/_opendistro/_alerting/monitors/",
		requestBody,
		getCommonHeaders(esClient))
	if err != nil {
		return errors.Wrap(err, "Unable to create new Monitor")
	}
	if resp.Status != 201 {
		indentJSON, _ := json.MarshalIndent(resp.Data, "", "\t")
		return errors.New("Unable to create monitor " + monitor.Name + string(indentJSON))
	}
	return nil
}

// Delete delete a monitor from remote
func (monitor *Monitor) Delete(esClient es.Client) error {
	var requestBody []byte
	resp, err := esClient.MakeRequest(http.MethodDelete,
		"/_opendistro/_alerting/monitors/"+monitor.id,
		requestBody,
		getCommonHeaders(esClient))
	if err != nil {
		return errors.Wrap(err, "Unable to delete a monitor "+monitor.Name)
	}
	if resp.Status != 200 {
		return errors.New("Unable to delete monitor" + monitor.Name + " ")
	}
	return nil
}
