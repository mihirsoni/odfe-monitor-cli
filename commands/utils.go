package commands

import (
	"net/url"

	"../monitor"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

// Skip IDs in comparision
func isIDKey(p cmp.Path) bool {
	step := p[len(p)-1].String()
	return step == ".ID"
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

func IsURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
