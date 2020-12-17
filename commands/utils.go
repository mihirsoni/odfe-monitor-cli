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
	"net/url"

	"github.com/autero1/odfe-monitor-cli/monitor"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func isMonitorChanged(localMonitor monitor.Monitor, remoteMonitor monitor.Monitor) bool {
	localYaml, _ := yaml.Marshal(localMonitor)
	remoteYml, _ := yaml.Marshal(remoteMonitor)
	diffs := dmp.DiffMain(string(remoteYml), string(localYaml), true)
	if len(diffs) > 1 {
		return true
	}
	return false
}

func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
