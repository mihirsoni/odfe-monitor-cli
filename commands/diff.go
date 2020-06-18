/*
 * Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License").
 * You may not use this file except in compliance with the License.
 * A copy of the License is located at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * or in the "license" file accompanying this file. This file is distributed
 * on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
 * express or implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

package commands

import (
	"fmt"
	"os"

	"github.com/mihirsoni/odfe-monitor-cli/destination"
	"github.com/mihirsoni/odfe-monitor-cli/monitor"

	mapset "github.com/deckarep/golang-set"
	"github.com/fatih/color"
	"github.com/sergi/go-diff/diffmatchpatch"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var diff = &cobra.Command{
	Use:   "diff",
	Short: "Show changes between remote and local monitors",
	Long:  `This command will show difference between remote and local monitors.`,
	Run:   showDiff,
}
var dmp = diffmatchpatch.New()

func showDiff(cmd *cobra.Command, args []string) {
	destinations, err := destination.GetRemote(esClient)
	check(err)
	localMonitors, localMonitorSet, err := monitor.GetAllLocal(rootDir)
	check(err)
	if localMonitorSet.Cardinality() == 0 {
		log.Info("There are no monitors")
		os.Exit(1)
	}
	allRemoteMonitors, remoteMonitorsSet, err := monitor.GetAllRemote(esClient, destinations)
	check(err)
	unTrackedMonitors := remoteMonitorsSet.Difference(localMonitorSet)
	allNewMonitors := localMonitorSet.Difference(remoteMonitorsSet)
	allCommonMonitors := remoteMonitorsSet.Intersect(localMonitorSet)

	changedMonitors := mapset.NewSet()
	allCommonMonitorsIt := allCommonMonitors.Iterator()
	for commonMonitor := range allCommonMonitorsIt.C {
		monitorName := commonMonitor.(string)
		if isMonitorChanged(localMonitors[monitorName], allRemoteMonitors[monitorName]) == true {
			changedMonitors.Add(commonMonitor)
		}
	}
	hasDeleted := unTrackedMonitors.Cardinality() > 0
	hasModified := changedMonitors.Cardinality() > 0
	hasCreated := allNewMonitors.Cardinality() > 0
	//All New Monitors
	if hasCreated {
		log.Debug("New monitors to be publushed", allNewMonitors)
		fmt.Println("---------------------------------------------------------")
		fmt.Println(" These monitors are currently missing in alerting ")
		fmt.Println("---------------------------------------------------------")
		for newMonitor := range allNewMonitors.Iterator().C {
			monitorName := newMonitor.(string)
			localYaml, _ := yaml.Marshal(localMonitors[monitorName])
			color.Green(string(localYaml))
		}
	}

	if hasModified {
		log.Debug("Common monitors to be updated", allCommonMonitors)
		fmt.Println("---------------------------------------------------------")
		fmt.Println(" These are existing monitors, which have been modified ")
		fmt.Println("---------------------------------------------------------")
		for monitorToBeUpdated := range changedMonitors.Iterator().C {
			monitorName := monitorToBeUpdated.(string)
			localYaml, err := yaml.Marshal(localMonitors[monitorName])
			check(err)
			remoteYml, err := yaml.Marshal(allRemoteMonitors[monitorName])
			check(err)
			diffs := dmp.DiffMain(string(remoteYml), string(localYaml), true)
			diffs = dmp.DiffCleanupSemantic(diffs)
			fmt.Println(dmp.DiffPrettyText(diffs))
		}
	}
	if hasDeleted {
		log.Debug("Un-tracked monitors to be deleted", unTrackedMonitors)
		fmt.Println("---------------------------------------------------------")
		fmt.Println(" These monitors will be deleted if push with the --delete flag")
		fmt.Println("---------------------------------------------------------")
		for newMonitor := range unTrackedMonitors.Iterator().C {
			monitorName := newMonitor.(string)
			remoteYml, _ := yaml.Marshal(allRemoteMonitors[monitorName])
			color.Red(string(remoteYml))
		}
	}
}

func init() {
	rootCmd.AddCommand(diff)
}
