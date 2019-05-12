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

// Monitor nice
type Monitor struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Enabled  bool     `json:"enabled"`
	Schedule Schedule `json:"schedule"`
}

func (monitor *Monitor) getMonitor() *Monitor {
	yamlFile, err := ioutil.ReadFile("monitor.yml")
	if err != nil {
		fmt.Println("Unable to parse monitor file", err)
		os.Exit(1)
	}
	err = yaml.Unmarshal(yamlFile, &monitor)
	if err != nil {
		fmt.Println("Unable to parse the yml file", err)
		os.Exit(1)
	}
	return monitor
}
func main() {
	// flag.Parse()
	// if flag.NFlag() == 0 {
	// 	fmt.Printf("Usage: %s [options] \n", os.Args[0])
	// 	fmt.Println("options")
	// 	flag.PrintDefaults()
	// 	os.Exit(1)
	// }
	// monitor := &Monitor{
	// 	Name:    "Test",
	// 	Type:    "monitor",
	// 	Enabled: true,
	// 	Schedule: Schedule{
	// 		Period: Period{
	// 			Interval: 1,
	// 			Unit:     "Min",
	// 		},
	// 	},
	// }
	var monitor Monitor
	monitor.getMonitor()
	jso, _ := json.Marshal(monitor)
	fmt.Println("monitor is ", string(jso))

	allMonitors := getRemoteMonitors()

	localYaml, err := yaml.Marshal(&monitor)
	remoteYml, err := yaml.Marshal(allMonitors[0])
	if err != nil {
		fmt.Printf("Unable to convert into YML")
		os.Exit(1)
	}
	//Test end
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(localYaml), string(remoteYml), true)
	fmt.Println(len(diffs))
	fmt.Println(dmp.DiffPrettyText(diffs))
	fmt.Println(len(allMonitors))
}

func init() {
	flag.StringVarP(&user, "user", "u", "", "Search Users")
}

func getLocalMonitors() []Monitor {
	return nil
}

func getRemoteMonitors() []Monitor {
	var (
		r           map[string]interface{}
		allMonitors []Monitor
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
		allMonitors = append(allMonitors, monitor)
		// tst, err := yaml.Marshal(&monitor)
		// if err != nil {
		// 	fmt.Printf("Unable to convert into YML")
		// 	os.Exit(1)
		// }
		// fmt.Println(string(tst))
		// fmt.Println(monitor.Schedule.Period.Unit)
	}
	// fmt.Println(len(allMonitors))
	return allMonitors
}

func getLocalMonitors() {

}
