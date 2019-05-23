package commands

import (
	"strconv"

	"../destination"
	"../monitor"
	mapset "github.com/deckarep/golang-set"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var push = &cobra.Command{
	Use:   "push [Options]",
	Short: "push all changed to remote Elasticsearch",
	Long:  `This command will push all the updated changes to elasticsearch cluster. Be careful on while passing --delete flag , there is no way to bring them back unless you've snapshot`,
	Run:   runPush,
}

var deleteUnTracked bool

func runPush(cmd *cobra.Command, args []string) {
	destinations, err := destination.GetLocal(rootDir)
	check(err)

	localMonitors, localMonitorSet, err := monitor.GetAllLocal(rootDir)
	if err != nil {
		log.Fatal("Unable to parse monitors from yaml files due to ", err)
	}
	remoteMonitors, remoteMonitorsSet, err := monitor.GetAllRemote(Config, destinations)
	check(err)

	unTrackedMonitors := remoteMonitorsSet.Difference(localMonitorSet)
	cliNewMonitors := localMonitorSet.Difference(remoteMonitorsSet)
	cliManagedMonitors := remoteMonitorsSet.Intersect(localMonitorSet)

	modifiedMonitors := mapset.NewSet()
	cliManagedMonitorsIt := cliManagedMonitors.Iterator()
	for cliManaged := range cliManagedMonitorsIt.C {
		if isMonitorChanged(localMonitors[cliManaged.(string)], remoteMonitors[cliManaged.(string)]) == true {
			modifiedMonitors.Add(cliManaged)
		}
	}

	monitorsToBeUpdated := cliNewMonitors.Union(modifiedMonitors)
	shouldDelete := deleteUnTracked && unTrackedMonitors.Cardinality() > 0
	shouldUpdate := modifiedMonitors.Cardinality() > 0
	shouldCreate := cliNewMonitors.Cardinality() > 0
	if !shouldCreate && !shouldUpdate && !shouldDelete {
		log.Info("All monitors are up-to-date with remote monitors")
		return
	}

	var preparedMonitors map[string]monitor.Monitor
	preparedMonitors = make(map[string]monitor.Monitor)
	// RunAll monitor before making update to ensure they're valid
	for currentMonitor := range monitorsToBeUpdated.Iterator().C {
		monitorName := currentMonitor.(string)
		modifiedMonitor, err := monitor.Prepare(localMonitors[monitorName],
			remoteMonitors[monitorName],
			destinations,
			!cliNewMonitors.Contains(monitorName))
		check(err)
		preparedMonitors[monitorName] = modifiedMonitor
	}
	runMonitors(monitorsToBeUpdated, preparedMonitors)
	if shouldUpdate {
		log.Debug("Monitors to be updated in remote ", modifiedMonitors)
		updateMonitors(modifiedMonitors, remoteMonitors, preparedMonitors)
	}
	if shouldCreate {
		log.Debug("Monitors to be created in remote", unTrackedMonitors)
		createMonitors(cliNewMonitors, preparedMonitors)
	}
	if shouldDelete {
		log.Debug("Monitors to be deleted from remote", unTrackedMonitors)
		deleteMonitors(unTrackedMonitors, remoteMonitors)
	}
}

func updateMonitors(
	updateMonitors mapset.Set,
	remoteMonitors map[string]monitor.Monitor,
	preparedMonitors map[string]monitor.Monitor) {

	updateMonitorCh := make(chan error)
	for newMonitor := range updateMonitors.Iterator().C {
		monitorName := newMonitor.(string)
		go monitor.Update(Config, remoteMonitors[monitorName], preparedMonitors[monitorName], updateMonitorCh)
	}
	successfulUpdates := 0
	for range updateMonitors.Iterator().C {
		err := <-updateMonitorCh
		if err != nil {
			log.Fatal(err)
		} else {
			successfulUpdates++
		}
	}
	log.Print("Updated " + strconv.Itoa(successfulUpdates) + "/" + strconv.Itoa(updateMonitors.Cardinality()) + " monitors")
}

func createMonitors(newMonitors mapset.Set, preparedMonitors map[string]monitor.Monitor) {
	createMonitorCh := make(chan error)
	for newMonitor := range newMonitors.Iterator().C {
		monitorName := newMonitor.(string)
		go monitor.Create(Config, preparedMonitors[monitorName], createMonitorCh)
	}
	successfulUpdates := 0
	for range newMonitors.Iterator().C {

		err := <-createMonitorCh
		if err != nil {
			log.Fatal(err)
		} else {
			successfulUpdates++
		}
	}
	log.Print("Created " + strconv.Itoa(successfulUpdates) + "/" + strconv.Itoa(newMonitors.Cardinality()) + " monitors")
}

func runMonitors(monitorsToBeUpdated mapset.Set, preparedMonitors map[string]monitor.Monitor) {
	// RunAll monitor before making update to ensure they're valid
	runChan := make(chan error)
	for currentMonitor := range monitorsToBeUpdated.Iterator().C {
		monitorName := currentMonitor.(string)
		go monitor.Run(Config, preparedMonitors[monitorName], runChan)
	}
	for range monitorsToBeUpdated.Iterator().C {
		err := <-runChan
		if err != nil {
			log.Fatal(err)
		}
	}
}

func deleteMonitors(monitorsToBeDeleted mapset.Set, remoteMonitors map[string]monitor.Monitor) {
	chDeleteMonitor := make(chan error)
	for currentMonitor := range monitorsToBeDeleted.Iterator().C {
		monitorName := currentMonitor.(string)
		go monitor.Delete(Config, remoteMonitors[monitorName], chDeleteMonitor)
	}
	successfulDelete := 0
	for range monitorsToBeDeleted.Iterator().C {
		err := <-chDeleteMonitor
		if err != nil {
			log.Fatal(err)
		} else {
			successfulDelete++
		}
	}
	log.Print("Deleted " + strconv.Itoa(successfulDelete) + "/" + strconv.Itoa(monitorsToBeDeleted.Cardinality()) + " monitors")
}

func init() {
	push.Flags().BoolVar(&deleteUnTracked, "delete", false, "delete un-tracked monitors from remote es cluster")
	rootCmd.AddCommand(push)

}
