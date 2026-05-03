// cmd/clickup/helpers.go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hiraking/click-up-cli/internal/config"
)

func loadConfig() (*config.AppConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return config.Load(filepath.Join(home, ".clickup", "config.json"))
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func availableListNames(lists map[string]string) string {
	names := make([]string, 0, len(lists))
	for k := range lists {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
