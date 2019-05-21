package commands

import (
	"fmt"
	"os"

	"../destination"
	"../monitor"
	mapset "github.com/deckarep/golang-set"
	"github.com/fatih/color"
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

	localMonitors, localMonitorSet, err := monitor.GetAllLocal(rootDir)
	if err != nil {
		fmt.Println("Unable to parse monitors from yaml files due to ", err)
		os.Exit(1)
	}
	remoteMonitors, remoteMonitorsSet, err := monitor.GetAllRemote(Config, destinations)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//TODO:: May be delete un tracked ?
	// unTrackedMonitors := remoteMonitorsSet.Difference(localMonitorSet)
	cliNewMonitors := localMonitorSet.Difference(remoteMonitorsSet)
	cliManagedMonitors := remoteMonitorsSet.Intersect(localMonitorSet)
	fmt.Println("All new monitor", cliNewMonitors)
	fmt.Println("All common monitors", cliManagedMonitors)

	modifiedMonitors := mapset.NewSet()
	cliManagedMonitorsIt := cliManagedMonitors.Iterator()
	for cliManaged := range cliManagedMonitorsIt.C {
		if isMonitorChanged(localMonitors[cliManaged.(string)], remoteMonitors[cliManaged.(string)]) == true {
			modifiedMonitors.Add(cliManaged)
		}
	}
	monitorsToBeUpdated := cliNewMonitors.Union(modifiedMonitors)
	fmt.Println("All common monitors", monitorsToBeUpdated)
	if monitorsToBeUpdated.Cardinality() == 0 {
		color.Green("No monitors to update")
		os.Exit(1)
	}

	var preparedMonitors map[string]monitor.Monitor
	preparedMonitors = make(map[string]monitor.Monitor)
	// RunAll monitor before making update to ensure they're valid
	runChan := make(chan error)
	for currentMonitor := range monitorsToBeUpdated.Iterator().C {
		monitorName := currentMonitor.(string)
		modifiedMonitor := monitor.Prepare(localMonitors[monitorName],
			remoteMonitors[monitorName],
			destinations,
			!cliNewMonitors.Contains(monitorName))
		//Run monitor
		preparedMonitors[monitorName] = modifiedMonitor
		go monitor.Run(Config, modifiedMonitor, runChan)
	}
	for range monitorsToBeUpdated.Iterator().C {
		err = <-runChan
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	createOrUpdate := make(chan error)
	for currentMonitor := range monitorsToBeUpdated.Iterator().C {
		monitorName := currentMonitor.(string)
		isNewMonitor := cliNewMonitors.Contains(monitorName)
		if isNewMonitor {
			monitor.Create(Config, preparedMonitors[monitorName], createOrUpdate)
		} else {
			monitor.Update(Config, remoteMonitors[monitorName], preparedMonitors[monitorName], createOrUpdate)
		}
	}
	successfulUpdates := 0
	for range monitorsToBeUpdated.Iterator().C {
		err = <-createOrUpdate
		fmt.Println("Hello", err)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Hello")
			successfulUpdates++
		}
	}
	color.Green("Successfully created / updated monitors ", string(successfulUpdates))
	os.Exit(1)
}

func init() {
	RootCmd.AddCommand(push)
}
