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
	remoteMonitors, remoteMonitorsSet := monitor.GetRemoteMonitors(ESConfig)
	// unTrackedMonitors := remoteMonitorsSet.Difference(localMonitorSet)
	// fmt.Println("All un tracked monitor", unTrackedMonitors)
	cliNewMonitors := localMonitorSet.Difference(remoteMonitorsSet)
	cliManagedMonitors := remoteMonitorsSet.Intersect(localMonitorSet)
	fmt.Println("All new monitor", cliNewMonitors)
	fmt.Println("All common monitors", cliManagedMonitors)

	modifiedMonitors := mapset.NewSet()
	cliManagedMonitorsIt := cliManagedMonitors.Iterator()
	for cliManaged := range cliManagedMonitorsIt.C {
		if isMonitorChanged(localMonitors[cliManaged.(string)], remoteMonitors[cliManaged.(string)]) != true {
			modifiedMonitors.Add(cliManaged)
		}
	}
	monitorsToBeUpdated := cliNewMonitors.Union(modifiedMonitors)
	fmt.Println("All common monitors", monitorsToBeUpdated)
	var preparedMonitors map[string]monitor.Monitor
	preparedMonitors = make(map[string]monitor.Monitor)
	// RunAll monitor before making update to ensure they're valid
	for currentMonitor := range monitorsToBeUpdated.Iterator().C {
		monitorName := currentMonitor.(string)
		modifiedMonitor := monitor.PrepareMonitor(localMonitors[monitorName],
			remoteMonitors[monitorName],
			!cliNewMonitors.Contains(monitorName))
		//Run monitor
		_, err := monitor.RunMonitor(ESConfig, modifiedMonitor)
		if err != nil {
			fmt.Println("Unable to run the monitor "+monitorName, err)
			os.Exit(1)
		}
		preparedMonitors[monitorName] = modifiedMonitor
	}
	for currentMonitor := range monitorsToBeUpdated.Iterator().C {
		monitorName := currentMonitor.(string)
		isNewMonitor := cliNewMonitors.Contains(monitorName)
		if isNewMonitor {
			monitor.CreateNewMonitor(ESConfig, preparedMonitors[monitorName])
		} else {
			monitor.UpdateMonitor(ESConfig, remoteMonitors[monitorName], preparedMonitors[monitorName])
		}
	}
	fmt.Println(len(remoteMonitors))
}

func init() {
	RootCmd.AddCommand(push)
}
