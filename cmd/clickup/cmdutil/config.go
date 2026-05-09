// cmd/clickup/cmdutil/config.go
package cmdutil

import (
	"os"
	"path/filepath"

	"github.com/hiraking/click-up-cli/internal/config"
)

// ResolveConfigPath resolves the config file path with the following priority:
//  1. configPath argument (explicit --config flag value)
//  2. CLICKUP_CONFIG environment variable
//  3. ~/.clickup/config.json (default, only if it exists)
//  4. "" (no file, env-var-only mode)
func ResolveConfigPath(configPath string) string {
	if configPath != "" {
		return configPath
	}
	if env := os.Getenv("CLICKUP_CONFIG"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	defaultPath := filepath.Join(home, ".clickup", "config.json")
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		return ""
	}
	return defaultPath
}

func LoadConfig(configPath string) (*config.AppConfig, error) {
	return config.Load(ResolveConfigPath(configPath))
}
