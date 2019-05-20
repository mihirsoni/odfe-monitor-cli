package commands

import (
	"fmt"
	"os"

	"../destination"
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
	destinations, err := destination.GetLocal(rootDir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//Push Monitors
	localMonitors, localMonitorSet, err := monitor.GetLocalMonitors(rootDir)
	if err != nil {
		fmt.Println("Unable to parse monitors from yaml files due to ", err)
		os.Exit(1)
	}
	remoteMonitors, remoteMonitorsSet := monitor.GetRemoteMonitors(Config, destinations)
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
			destinations,
			!cliNewMonitors.Contains(monitorName))
		//Run monitor
		_, err := monitor.RunMonitor(Config, modifiedMonitor)
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
			monitor.CreateNewMonitor(Config, preparedMonitors[monitorName])
		} else {
			monitor.UpdateMonitor(Config, remoteMonitors[monitorName], preparedMonitors[monitorName])
		}
	}
	fmt.Println(len(remoteMonitors))
}

func init() {
	RootCmd.AddCommand(push)
}
