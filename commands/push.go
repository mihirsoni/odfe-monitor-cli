package commands

import (
	"fmt"
	"os"

	"../monitor"
	mapset "github.com/deckarep/golang-set"
	"github.com/spf13/cobra"
)

var push = &cobra.Command{
	Use:   "push",
	Short: "push all changed to remote Elasticsearch",
	Long:  `This command will push all the updated changes to elasticsearch cluster`,
	Run:   runPush,
}

func runPush(cmd *cobra.Command, args []string) {
	//Push Monitors
	localMonitors, localMonitorSet := monitor.GetLocalMonitors()
	allRemoteMonitors, remoteMonitorsSet := monitor.GetRemoteMonitors(ESConfig)
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

	// RunAll monitor before making update to ensure they're valid
	for monitorToBeUpdated := range changedMonitors.Iterator().C {
		monitorName := monitorToBeUpdated.(string)
		canonicalMonitor := monitor.PrepareMonitor(localMonitors[monitorName], allRemoteMonitors[monitorName], true)
		_, err := monitor.RunMonitor(ESConfig, canonicalMonitor)
		if err != nil {
			fmt.Println("Unable to run the monitor "+canonicalMonitor.Name, err)
			os.Exit(1)
		}
	}

	// All monitors are verified hit update,
	for monitorToBeUpdated := range changedMonitors.Iterator().C {
		monitorName := monitorToBeUpdated.(string)
		canonicalMonitor := monitor.PrepareMonitor(localMonitors[monitorName], allRemoteMonitors[monitorName], true)
		monitor.UpdateMonitor(ESConfig, allRemoteMonitors[monitorName], canonicalMonitor)
	}
	allNewMonitorsIT := allNewMonitors.Iterator()
	for newMonitor := range allNewMonitorsIT.C {
		newMonitor := monitor.PrepareMonitor(localMonitors[newMonitor.(string)], monitor.Monitor{}, false)
		monitor.CreateNewMonitor(ESConfig, newMonitor)
	}
	fmt.Println(len(allRemoteMonitors))
}

func init() {
	RootCmd.AddCommand(push)
}
