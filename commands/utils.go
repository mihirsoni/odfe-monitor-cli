package commands

import (
	"net/url"

	"github.com/mihirsoni/odfe-monitor-cli/monitor"
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
