package commands

import (
	"fmt"
	"os"

	"../destination"
	"../monitor"
	mapset "github.com/deckarep/golang-set"
	"github.com/fatih/color"
	"github.com/sergi/go-diff/diffmatchpatch"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var diff = &cobra.Command{
	Use:   "diff",
	Short: "delta between local and remote monitors",
	Long:  `this will print diff between local and remote monitors.`,
	Run:   showDiff,
}
var dmp = diffmatchpatch.New()

func showDiff(cmd *cobra.Command, args []string) {
	destinations, err := destination.GetLocal(rootDir)
	check(err)
	localMonitors, localMonitorSet, err := monitor.GetAllLocal(rootDir)
	check(err)
	if localMonitorSet.Cardinality() == 0 {
		log.Info("There are no monitors")
		os.Exit(1)
	}
	allRemoteMonitors, remoteMonitorsSet, err := monitor.GetAllRemote(Config, destinations)
	check(err)
	unTrackedMonitors := remoteMonitorsSet.Difference(localMonitorSet)
	allNewMonitors := localMonitorSet.Difference(remoteMonitorsSet)
	allCommonMonitors := remoteMonitorsSet.Intersect(localMonitorSet)
	if Verbose {
		log.Debug("Un tracked monitors", unTrackedMonitors)
		log.Debug("New monitors", allNewMonitors)
		log.Debug("Common monitors", allCommonMonitors)
	}
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
			check(err)
			remoteYml, err := yaml.Marshal(allRemoteMonitors[monitorName])
			check(err)
			diffs := dmp.DiffMain(string(remoteYml), string(localYaml), true)
			diffs = dmp.DiffCleanupSemantic(diffs)
			fmt.Println(dmp.DiffPrettyText(diffs))
		}
	}
}

func init() {
	RootCmd.AddCommand(diff)
}
