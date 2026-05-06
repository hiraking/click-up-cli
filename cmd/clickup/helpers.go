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
	return config.Load(resolveConfigPath())
}

// resolveConfigPath はconfig fileのパスを次の優先順位で決定する:
//  1. --config フラグ（明示的・ファイル必須）
//  2. CLICKUP_CONFIG 環境変数（明示的・ファイル必須）
//  3. ~/.clickup/config.json（デフォルト・存在する場合のみ）
//  4. "" （ファイルなし・env var のみで動作）
func resolveConfigPath() string {
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

func maskAPIKey(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}
