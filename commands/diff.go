package commands

import (
	"fmt"
	"os"

	"../monitor"
	mapset "github.com/deckarep/golang-set"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
	"gopkg.in/mihirsoni/yaml.v2"
)

var diff = &cobra.Command{
	Use:   "diff",
	Short: "difference between local and remote monitors",
	Long:  `this will show print diff between local and remote monitors.`,
	Run:   showDiff,
}

func showDiff(cmd *cobra.Command, args []string) {
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
		localYaml, err := yaml.Marshal(localMonitors[monitorName])
		remoteYml, err := yaml.Marshal(allRemoteMonitors[monitorName])
		fmt.Println(string(remoteYml))
		fmt.Println("-------------------------------")
		fmt.Println(string(localYaml))
		fmt.Println("-------------------------------")

		if err != nil {
			fmt.Printf("Unable to convert into YML")
			os.Exit(1)
		}
		dmp := diffmatchpatch.New()

		diffs := dmp.DiffMain(string(remoteYml), string(localYaml), false)
		fmt.Println(dmp.DiffPrettyText(diffs))
		// diff := cmp.Diff(allRemoteMonitors[monitorName], localMonitors[monitorName], cmpopts.IgnoreUnexported(monitor.Monitor{}))
		// fmt.Println(string(diff))
	}
}

func init() {
	RootCmd.AddCommand(diff)
}
