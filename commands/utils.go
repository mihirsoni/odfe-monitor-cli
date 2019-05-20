package commands

import (
	"fmt"
	"net/url"
	"reflect"

	"../monitor"
	"github.com/google/go-cmp/cmp"
)

// Skip IDs in comparision
func isIDKey(p cmp.Path) bool {
	step := p[len(p)-1].String()
	return step == ".ID"
}

func isMonitorChanged(localMonitor monitor.Monitor, remoteMonitor monitor.Monitor) bool {
	// diff := cmp.Diff(remoteMonitor, localMonitor, cmpopts.IgnoreUnexported(monitor.Monitor{}), cmp.FilterPath(isIDKey, cmp.Ignore()))
	// fmt.Println(string(diff))
	// return cmp.Equal(localMonitor, remoteMonitor, cmpopts.IgnoreUnexported(monitor.Monitor{}), cmp.FilterPath(isIDKey, cmp.Ignore()))
	fmt.Println("reflect.DeepEqual(remoteMonitor, localMonitor)", reflect.DeepEqual(remoteMonitor, localMonitor))
	return reflect.DeepEqual(remoteMonitor, localMonitor)
}

func IsURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
