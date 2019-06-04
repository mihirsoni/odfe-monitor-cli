package commands

import (
	mapset "github.com/deckarep/golang-set"
	"github.com/mihirsoni/odfe-alerting/destination"
	"github.com/mihirsoni/odfe-alerting/monitor"
	"github.com/mihirsoni/odfe-alerting/utils"
	log "github.com/sirupsen/logrus"
	"gopkg.in/cheggaaa/pb.v1"

	"github.com/spf13/cobra"
)

var push = &cobra.Command{
	Use:   "push",
	Short: "Publish all changed to remote Elasticsearch",
	Long:  `This command will push all modification to elasticsearch cluster. Be careful on while passing --delete flag, there is no way to bring them back unless you've snapshot`,
	Run:   runPush,
}

//Default parallel request for cluster
const (
	defaultLimit = 20
)

var deleteUnTracked bool
var dryRun bool
var submit bool

var bar *pb.ProgressBar

func init() {
	push.Flags().BoolVar(&deleteUnTracked, "delete", false, "Delete un-tracked monitors from remote es cluster")
	push.Flags().BoolVar(&dryRun, "dryRun", true, "Dry run monitor more detailed https://opendistro.github.io/for-elasticsearch-docs/docs/alerting/api/#run-monitor")
	push.Flags().BoolVar(&submit, "submit", false, "Publish monitors to remote")
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
			!cliNewMonitors.Contains(monitorName), esClient.OdVersion)
		check(err)
		preparedMonitors[monitorName] = localMonitor
	}
	// RunAll monitor before making update to ensure they're valid
	// All of these operations are sequential. Run, Modify, Create, Delete
	if shouldCreate || shouldUpdate {
		runMonitors(monitorsToBeUpdated, preparedMonitors)
		if !submit {
			log.Info("Use --submit argument to publish monitors")
			return
		}
	}
	if shouldUpdate {
		log.Debug("Monitors to be updated in remote ", modifiedMonitors)
		updateMonitors(modifiedMonitors, remoteMonitors, preparedMonitors)
	}
	if shouldCreate {
		log.Debug("Monitors to be created in remote ", cliNewMonitors)
		createMonitors(cliNewMonitors, preparedMonitors)
	}
	if shouldDelete {
		log.Debug("Monitors to be deleted from remote ", unTrackedMonitors)
		deleteMonitors(unTrackedMonitors, remoteMonitors)
	}
	log.Info("Done")
}

func updateMonitors(
	updateMonitors mapset.Set,
	remoteMonitors map[string]monitor.Monitor,
	preparedMonitors map[string]monitor.Monitor) {

	limiter := utils.NewLimiter(defaultLimit)
	successfulUpdates := 0
	if !Verbose {
		bar = pb.StartNew(updateMonitors.Cardinality())
		bar.Prefix("Updating monitors")
	}
	for newMonitor := range updateMonitors.Iterator().C {
		monitorName := newMonitor.(string)
		limiter.Execute(func() {
			log.Debug("Updating monitor: ", monitorName)
			currentMonitor := preparedMonitors[monitorName]
			err := currentMonitor.Update(esClient)
			if err == nil {
				successfulUpdates++
			} else {
				log.Debug(err)
			}
			if !Verbose {
				bar.Increment()
			}
		})
	}
	limiter.Wait()
	if !Verbose {
		bar.Finish()
	}
}

func createMonitors(newMonitors mapset.Set, preparedMonitors map[string]monitor.Monitor) {
	limiter := utils.NewLimiter(defaultLimit)
	successfullCreate := 0
	if !Verbose {
		bar = pb.StartNew(newMonitors.Cardinality())
		bar.Prefix("Creating monitors")
	}
	for newMonitor := range newMonitors.Iterator().C {
		monitorName := newMonitor.(string)
		limiter.Execute(func() {
			log.Debug("Creating monitor: ", monitorName)
			newMonitor := preparedMonitors[monitorName]
			err := newMonitor.Create(esClient)
			if err == nil {
				successfullCreate++
			} else {
				log.Debug(err)
			}
			if !Verbose {
				bar.Increment()
			}
		})
	}
	limiter.Wait()
	if !Verbose {
		bar.Finish()
	}
}

func runMonitors(monitorsToBeUpdated mapset.Set, preparedMonitors map[string]monitor.Monitor) {
	limiter := utils.NewLimiter(defaultLimit)
	if !Verbose {
		bar = pb.StartNew(monitorsToBeUpdated.Cardinality())
		bar.Prefix("Running monitors")
	}
	for currentMonitor := range monitorsToBeUpdated.Iterator().C {
		monitorName := currentMonitor.(string)
		limiter.Execute(func() {
			log.Debug("Running monitor: ", monitorName)
			runMonitor := preparedMonitors[monitorName]
			err := runMonitor.Run(esClient, dryRun)
			check(err)
			if !Verbose {
				bar.Increment()
			}
		})
	}
	limiter.Wait()
	if !Verbose {
		bar.Finish()
	}
}

func deleteMonitors(monitorsToBeDeleted mapset.Set, remoteMonitors map[string]monitor.Monitor) {
	limiter := utils.NewLimiter(defaultLimit)
	successfulDelete := 0
	if !Verbose {
		bar = pb.StartNew(monitorsToBeDeleted.Cardinality())
		bar.Prefix("Deleting monitors")
	}
	for currentMonitor := range monitorsToBeDeleted.Iterator().C {
		monitorName := currentMonitor.(string)
		remoteMonitor := remoteMonitors[monitorName]
		limiter.Execute(func() {
			err := remoteMonitor.Delete(esClient)
			if err == nil {
				successfulDelete++
			}
			if !Verbose {
				bar.Increment()
			}
		})
	}
	limiter.Wait()
	if !Verbose {
		bar.Finish()
	}
}
