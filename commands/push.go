package commands

import (
	"strconv"

	"../destination"
	"../monitor"
	"../utils"
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

const (
	DEFAULT_LIMIT = 20
)

var deleteUnTracked bool

func init() {
	push.Flags().BoolVar(&deleteUnTracked, "delete", false, "delete un-tracked monitors from remote es cluster")
	rootCmd.AddCommand(push)

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
	for currentMonitor := range monitorsToBeUpdated.Iterator().C {
		monitorName := currentMonitor.(string)
		modifiedMonitor, err := monitor.Prepare(localMonitors[monitorName],
			remoteMonitors[monitorName],
			destinations,
			!cliNewMonitors.Contains(monitorName))
		check(err)
		preparedMonitors[monitorName] = modifiedMonitor
	}
	// RunAll monitor before making update to ensure they're valid
	// All of these operations are sequential. Run, Modify, Create, Delete
	runMonitors(monitorsToBeUpdated, preparedMonitors)
	if shouldUpdate {
		log.Debug("Monitors to be updated in remote ", modifiedMonitors)
		updateMonitors(modifiedMonitors, remoteMonitors, preparedMonitors)
	}
	if shouldCreate {
		log.Debug("Monitors to be created in remote", cliNewMonitors)
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

	limiter := utils.NewLimiter(DEFAULT_LIMIT)
	successfulUpdates := 0
	for newMonitor := range updateMonitors.Iterator().C {
		monitorName := newMonitor.(string)
		limiter.Execute(func() {
			log.Debug("Updating monitor: ", monitorName)
			err := monitor.Update(Config, remoteMonitors[monitorName], preparedMonitors[monitorName])
			if err == nil {
				successfulUpdates++
			} else {
				log.Debug(err)
			}
		})
	}
	limiter.Wait()
	log.Print("Updated " + strconv.Itoa(successfulUpdates) + "/" + strconv.Itoa(updateMonitors.Cardinality()) + " monitors")
}

func createMonitors(newMonitors mapset.Set, preparedMonitors map[string]monitor.Monitor) {
	limiter := utils.NewLimiter(DEFAULT_LIMIT)
	successfullCreate := 0
	for newMonitor := range newMonitors.Iterator().C {
		monitorName := newMonitor.(string)
		limiter.Execute(func() {
			log.Debug("Creating monitor: ", monitorName)
			err := monitor.Create(Config, preparedMonitors[monitorName])
			if err == nil {
				successfullCreate++
			} else {
				log.Debug(err)
			}
		})
	}
	limiter.Wait()
	log.Print("Created " + strconv.Itoa(successfullCreate+1) + "/" + strconv.Itoa(newMonitors.Cardinality()) + " monitors")
}

func runMonitors(monitorsToBeUpdated mapset.Set, preparedMonitors map[string]monitor.Monitor) {
	limiter := utils.NewLimiter(DEFAULT_LIMIT)
	for currentMonitor := range monitorsToBeUpdated.Iterator().C {
		monitorName := currentMonitor.(string)
		limiter.Execute(func() {
			log.Debug("Running monitor: ", monitorName)
			err := monitor.Run(Config, preparedMonitors[monitorName])
			check(err)
		})
	}
	limiter.Wait()
}

func deleteMonitors(monitorsToBeDeleted mapset.Set, remoteMonitors map[string]monitor.Monitor) {
	limiter := utils.NewLimiter(DEFAULT_LIMIT)
	successfulDelete := 0
	for currentMonitor := range monitorsToBeDeleted.Iterator().C {
		monitorName := currentMonitor.(string)
		limiter.Execute(func() {
			err := monitor.Delete(Config, remoteMonitors[monitorName])
			if err == nil {
				successfulDelete++
			}
		})
	}
	limiter.Wait()
	log.Print("Deleted " + strconv.Itoa(successfulDelete+1) + "/" + strconv.Itoa(monitorsToBeDeleted.Cardinality()) + " monitors")
}
