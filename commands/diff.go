package commands

import (
	"fmt"
	"os"

	"../destination"
	"../monitor"
	mapset "github.com/deckarep/golang-set"
	"github.com/fatih/color"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var diff = &cobra.Command{
	Use:   "diff",
	Short: "difference between local and remote monitors",
	Long:  `this will show print diff between local and remote monitors.`,
	Run:   showDiff,
}
var dmp = diffmatchpatch.New()

func isMonitorChanged(localMonitor monitor.Monitor, remoteMonitor monitor.Monitor) bool {
	localYaml, _ := yaml.Marshal(localMonitor)
	remoteYml, _ := yaml.Marshal(remoteMonitor)
	diffs := dmp.DiffMain(string(remoteYml), string(localYaml), true)
	fmt.Println("diffs", len(diffs))
	if len(diffs) > 1 {
		return true
	}
	return false
}

func showDiff(cmd *cobra.Command, args []string) {
	destinations, err := destination.GetLocal(rootDir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	localMonitors, localMonitorSet, err := monitor.GetAllLocal(rootDir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	allRemoteMonitors, remoteMonitorsSet, err := monitor.GetAllRemote(Config, destinations)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	unTrackedMonitors := remoteMonitorsSet.Difference(localMonitorSet)
	allNewMonitors := localMonitorSet.Difference(remoteMonitorsSet)
	allCommonMonitors := remoteMonitorsSet.Intersect(localMonitorSet)
	fmt.Println("All un tracked monitor", unTrackedMonitors)
	fmt.Println("All new monitor", allNewMonitors)
	fmt.Println("All common monitors", allCommonMonitors)
	changedMonitors := mapset.NewSet()
	allCommonMonitorsIt := allCommonMonitors.Iterator()
	for commonMonitor := range allCommonMonitorsIt.C {
		monitorName := commonMonitor.(string)
		if isMonitorChanged(localMonitors[monitorName], allRemoteMonitors[monitorName]) == true {
			changedMonitors.Add(commonMonitor)
		}
	}
	//All New Monitors
	if allNewMonitors.Cardinality() > 0 {
		fmt.Println("---------------------------------------------------------")
		fmt.Println(" These monitors are currently missing in alerting ")
		fmt.Println("---------------------------------------------------------")
		for newMonitor := range allNewMonitors.Iterator().C {
			monitorName := newMonitor.(string)
			localYaml, _ := yaml.Marshal(localMonitors[monitorName])
			color.Green(string(localYaml))
		}
	}

	if changedMonitors.Cardinality() > 0 {
		fmt.Println("---------------------------------------------------------")
		fmt.Println(" These are existing monitors, which have been modified ")
		fmt.Println("---------------------------------------------------------")
		for monitorToBeUpdated := range changedMonitors.Iterator().C {
			monitorName := monitorToBeUpdated.(string)
			localYaml, err := yaml.Marshal(localMonitors[monitorName])
			remoteYml, err := yaml.Marshal(allRemoteMonitors[monitorName])
			if err != nil {
				fmt.Printf("Unable to convert into YML")
				os.Exit(1)
			}
			diffs := dmp.DiffMain(string(remoteYml), string(localYaml), true)
			diffs = dmp.DiffCleanupSemantic(diffs)
			fmt.Println(dmp.DiffPrettyText(diffs))
		}
	}
}

func init() {
	RootCmd.AddCommand(diff)
}
