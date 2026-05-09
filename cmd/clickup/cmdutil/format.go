// cmd/clickup/cmdutil/format.go
package cmdutil

import (
	"sort"
	"strings"
)

func AvailableListNames(lists map[string]string) string {
	names := make([]string, 0, len(lists))
	for k := range lists {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func MaskAPIKey(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}
