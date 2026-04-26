// cmd/clickup/helpers.go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/hiraking/click-up-client/internal/config"
)

func loadConfig() (*config.AppConfig, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return config.Load(filepath.Join(filepath.Dir(execPath), "config.json"))
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
