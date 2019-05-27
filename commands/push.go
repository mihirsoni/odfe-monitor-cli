package commands

import (
	"strconv"

	"../destination"
	"../monitor"
	"../utils"
	"gopkg.in/cheggaaa/pb.v1"

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

var bar *pb.ProgressBar

func init() {
	push.Flags().BoolVar(&deleteUnTracked, "delete", false, "delete un-tracked monitors from remote es cluster")
	rootCmd.AddCommand(push)
}

func runPush(cmd *cobra.Command, args []string) {
	destinations, err := destination.GetRemote(esClient)
	check(err)
	localMonitors, localMonitorSet, err := monitor.GetAllLocal(rootDir)
	if err != nil {
		log.Fatal("Unable to parse monitors from yaml files due to ", err)
	}
	remoteMonitors, remoteMonitorsSet, err := monitor.GetAllRemote(esClient, destinations)
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
		localMonitor := localMonitors[monitorName]
		err := localMonitor.Prepare(remoteMonitors[monitorName],
			destinations,
			!cliNewMonitors.Contains(monitorName))
		check(err)
		preparedMonitors[monitorName] = localMonitor
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
	log.Info("Done")
}

func updateMonitors(
	updateMonitors mapset.Set,
	remoteMonitors map[string]monitor.Monitor,
	preparedMonitors map[string]monitor.Monitor) {

	limiter := utils.NewLimiter(DEFAULT_LIMIT)
	successfulUpdates := 0
	if !Verbose {
		bar = pb.StartNew(updateMonitors.Cardinality())
		bar.Prefix("Updating monitors")
	}
	for newMonitor := range updateMonitors.Iterator().C {
		monitorName := newMonitor.(string)
		limiter.Execute(func() {
			log.Debug("Updating monitor: ", monitorName)
			if !Verbose {
				bar.Increment()
			}
			currentMonitor := preparedMonitors[monitorName]
			err := currentMonitor.Update(esClient)
			if err == nil {
				successfulUpdates++
			} else {
				log.Debug(err)
			}
		})
	}
	limiter.Wait()
	if !Verbose {
		bar.Finish()
	}
}

func createMonitors(newMonitors mapset.Set, preparedMonitors map[string]monitor.Monitor) {
	limiter := utils.NewLimiter(DEFAULT_LIMIT)
	successfullCreate := 0
	if !Verbose {
		bar = pb.StartNew(newMonitors.Cardinality())
		bar.Prefix("Creating monitors")
	}
	for newMonitor := range newMonitors.Iterator().C {
		monitorName := newMonitor.(string)
		limiter.Execute(func() {
			if !Verbose {
				bar.Increment()
			}
			log.Debug("Creating monitor: ", monitorName)
			newMonitor := preparedMonitors[monitorName]
			err := newMonitor.Create(esClient)
			if err == nil {
				successfullCreate++
			} else {
				log.Debug(err)
			}
		})
	}
	limiter.Wait()
	if !Verbose {
		bar.Finish()
	}
}

func runMonitors(monitorsToBeUpdated mapset.Set, preparedMonitors map[string]monitor.Monitor) {
	limiter := utils.NewLimiter(DEFAULT_LIMIT)
	if !Verbose {
		bar = pb.StartNew(monitorsToBeUpdated.Cardinality())
		bar.Prefix("Running monitors")
	}
	for currentMonitor := range monitorsToBeUpdated.Iterator().C {
		limiter.Execute(func() {
			monitorName := currentMonitor.(string)
			if !Verbose {
				bar.Increment()
			}
			log.Debug("Running monitor: ", monitorName)
			runMonitor := preparedMonitors[monitorName]
			err := runMonitor.Run(esClient)
			check(err)
		})
	}
	limiter.Wait()
	if !Verbose {
		bar.Finish()
	}
}

func deleteMonitors(monitorsToBeDeleted mapset.Set, remoteMonitors map[string]monitor.Monitor) {
	limiter := utils.NewLimiter(DEFAULT_LIMIT)
	successfulDelete := 0
	for currentMonitor := range monitorsToBeDeleted.Iterator().C {
		monitorName := currentMonitor.(string)
		remoteMonitor := remoteMonitors[monitorName]
		limiter.Execute(func() {
			err := remoteMonitor.Delete(esClient)
			if err == nil {
				successfulDelete++
			}
		})
	}
	limiter.Wait()
	log.Print("Deleted " + strconv.Itoa(successfulDelete+1) + "/" + strconv.Itoa(monitorsToBeDeleted.Cardinality()) + " monitors")
}
