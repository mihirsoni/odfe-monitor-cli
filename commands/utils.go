package commands

import (
	"fmt"
	"net/url"

	"../monitor"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func isMonitorChanged(localMonitor monitor.Monitor, remoteMonitor monitor.Monitor) bool {
	diff := cmp.Diff(remoteMonitor, localMonitor, cmpopts.IgnoreUnexported(monitor.Monitor{}))
	fmt.Println(string(diff))
	return cmp.Equal(localMonitor, remoteMonitor, cmpopts.IgnoreUnexported(monitor.Monitor{}))
}

func IsURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
