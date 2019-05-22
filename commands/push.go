package commands

import (
	"../destination"
	"../monitor"
	mapset "github.com/deckarep/golang-set"
	log "github.com/sirupsen/logrus"
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
	check(err)

	localMonitors, localMonitorSet, err := monitor.GetAllLocal(rootDir)
	if err != nil {
		log.Fatal("Unable to parse monitors from yaml files due to ", err)
	}
	remoteMonitors, remoteMonitorsSet, err := monitor.GetAllRemote(Config, destinations)
	check(err)
	//TODO:: May be delete un tracked ?
	// unTrackedMonitors := remoteMonitorsSet.Difference(localMonitorSet)
	cliNewMonitors := localMonitorSet.Difference(remoteMonitorsSet)
	cliManagedMonitors := remoteMonitorsSet.Intersect(localMonitorSet)

	modifiedMonitors := mapset.NewSet()
	cliManagedMonitorsIt := cliManagedMonitors.Iterator()
	for cliManaged := range cliManagedMonitorsIt.C {
		if isMonitorChanged(localMonitors[cliManaged.(string)], remoteMonitors[cliManaged.(string)]) == true {
			modifiedMonitors.Add(cliManaged)
		}
	}
	if Verbose {
		log.Debug("New monitors", cliNewMonitors)
		log.Debug("Common monitors", cliManagedMonitors)
		log.Debug("Modified monitors list", modifiedMonitors)
	}

	monitorsToBeUpdated := cliNewMonitors.Union(modifiedMonitors)

	if monitorsToBeUpdated.Cardinality() == 0 {
		log.Info("No monitors to update")
		return
	}
	var preparedMonitors map[string]monitor.Monitor
	preparedMonitors = make(map[string]monitor.Monitor)
	// RunAll monitor before making update to ensure they're valid
	runChan := make(chan error)
	for currentMonitor := range monitorsToBeUpdated.Iterator().C {
		monitorName := currentMonitor.(string)
		modifiedMonitor, err := monitor.Prepare(localMonitors[monitorName],
			remoteMonitors[monitorName],
			destinations,
			!cliNewMonitors.Contains(monitorName))
		//Run monitor
		check(err)
		preparedMonitors[monitorName] = modifiedMonitor
		go monitor.Run(Config, modifiedMonitor, runChan)
	}
	for range monitorsToBeUpdated.Iterator().C {
		err = <-runChan
		if err != nil {
			log.Fatal(err)
		}
	}
	createOrUpdate := make(chan error)
	for currentMonitor := range monitorsToBeUpdated.Iterator().C {
		monitorName := currentMonitor.(string)
		isNewMonitor := cliNewMonitors.Contains(monitorName)
		if isNewMonitor {
			go monitor.Create(Config, preparedMonitors[monitorName], createOrUpdate)
		} else {
			go monitor.Update(Config, remoteMonitors[monitorName], preparedMonitors[monitorName], createOrUpdate)
		}
	}
	successfulUpdates := 0
	for range monitorsToBeUpdated.Iterator().C {
		err = <-createOrUpdate
		if err != nil {
			log.Fatal(err)
		} else {
			successfulUpdates++
		}
	}
	log.Print("Successfully created / updated monitors ", successfulUpdates)
}

func init() {
	RootCmd.AddCommand(push)
}
