package commands

import (
	"fmt"

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
	allRemoteMonitors, remoteMonitorsSet := monitor.GetRemoteMonitors()
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

		canonicalMonitor := monitor.PrepareMonitor(localMonitors[monitorName], allRemoteMonitors[monitorName])
		// monitor.RunMonitor(allRemoteMonitors[monitorName].ID, canonicalMonitor)
		monitor.UpdateMonitor(allRemoteMonitors[monitorName], canonicalMonitor)
	}
	fmt.Println(len(allRemoteMonitors))
}

func init() {
	RootCmd.AddCommand(push)
}
