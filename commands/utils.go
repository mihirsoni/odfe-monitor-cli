package commands

import (
	"net/url"

	"github.com/google/go-cmp/cmp"
)

// Skip IDs in comparision
func isIDKey(p cmp.Path) bool {
	step := p[len(p)-1].String()
	return step == ".ID"
}

func IsURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
