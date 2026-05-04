# GitHub Actions Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `apiKey`/`teamId` を環境変数（`CLICKUP_API_KEY`/`CLICKUP_TEAM_ID`）で渡せるようにし、設定ファイルパスを `--config` フラグまたは `CLICKUP_CONFIG` 環境変数で指定できるようにする。

**Architecture:** `config.Load()` を拡張して空パス（ファイルなしモード）と env var オーバーライドをサポート。`main.go` に `--config` persistent flag を追加。`helpers.go` の `loadConfig()` がパス優先順位チェーンを担当する。

**Tech Stack:** Go, github.com/spf13/cobra, github.com/spf13/viper, github.com/stretchr/testify

---

## File Map

| ファイル | 変更種別 | 内容 |
|---|---|---|
| `internal/config/config_test.go` | 新規作成 | env var サポートのテスト |
| `internal/config/config.go` | 修正 | 空パス対応 + env var オーバーライド |
| `cmd/clickup/main.go` | 修正 | `--config` persistent flag 追加 |
| `cmd/clickup/helpers.go` | 修正 | `resolveConfigPath()` 追加、`loadConfig()` 修正 |
| `README.md` | 修正 | 環境変数・`--config` フラグ・GitHub Actions 使用例を追記 |

---

## Task 1: `config.Load()` に env var オーバーライドを追加（TDD）

**Files:**
- Create: `internal/config/config_test.go`
- Modify: `internal/config/config.go`

---

- [ ] **Step 1: テストファイルを作成する**

`internal/config/config_test.go` を新規作成:

```go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hiraking/click-up-cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_EmptyPath_EnvVarsOnly(t *testing.T) {
	t.Setenv("CLICKUP_API_KEY", "pk_env_key")
	t.Setenv("CLICKUP_TEAM_ID", "env_team_id")

	cfg, err := config.Load("")
	require.NoError(t, err)
	assert.Equal(t, "pk_env_key", cfg.APIKey)
	assert.Equal(t, "env_team_id", cfg.TeamID)
	assert.Empty(t, cfg.Lists)
}

func TestLoad_EmptyPath_MissingEnvVars(t *testing.T) {
	t.Setenv("CLICKUP_API_KEY", "")
	t.Setenv("CLICKUP_TEAM_ID", "")

	_, err := config.Load("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apiKey is required")
}

func TestLoad_EnvVarOverridesFile_APIKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"apiKey":"file_key","teamId":"file_team"}`), 0600))

	t.Setenv("CLICKUP_API_KEY", "env_key")
	t.Setenv("CLICKUP_TEAM_ID", "")

	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "env_key", cfg.APIKey)
	assert.Equal(t, "file_team", cfg.TeamID)
}

func TestLoad_FileOnly_EnvVarsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path,
		[]byte(`{"apiKey":"file_key","teamId":"file_team","lists":{"work":"123"}}`), 0600))

	t.Setenv("CLICKUP_API_KEY", "")
	t.Setenv("CLICKUP_TEAM_ID", "")

	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "file_key", cfg.APIKey)
	assert.Equal(t, "file_team", cfg.TeamID)
	assert.Equal(t, map[string]string{"work": "123"}, cfg.Lists)
}
```

---

- [ ] **Step 2: テストを実行して失敗を確認する**

```
go test ./internal/config/...
```

Expected: `TestLoad_EmptyPath_EnvVarsOnly` と `TestLoad_EmptyPath_MissingEnvVars` が FAIL（`Load("")` がファイルなしで動かないため）。

---

- [ ] **Step 3: `config.Load()` を拡張する**

`internal/config/config.go` の `Load` 関数全体を次のコードに置き換える:

```go
// Load は設定を読み込む。path が空のときはファイルを読まず env var のみを使う。
// path が指定された場合はファイルを必須とする。
// CLICKUP_API_KEY / CLICKUP_TEAM_ID 環境変数はファイルの値を上書きする。
func Load(path string) (*AppConfig, error) {
	v := viper.New()

	if path != "" {
		v.SetConfigFile(path)

		if err := v.ReadInConfig(); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("config file not found: %w", os.ErrNotExist)
			}
			var pathErr *os.PathError
			if errors.As(err, &pathErr) {
				return nil, fmt.Errorf("config file not found: %w", os.ErrNotExist)
			}
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if key := os.Getenv("CLICKUP_API_KEY"); key != "" {
		cfg.APIKey = key
	}
	if team := os.Getenv("CLICKUP_TEAM_ID"); team != "" {
		cfg.TeamID = team
	}

	if cfg.APIKey == "" {
		return nil, errors.New("config: apiKey is required")
	}
	if cfg.TeamID == "" {
		return nil, errors.New("config: teamId is required")
	}

	return &cfg, nil
}
```

---

- [ ] **Step 4: テストを実行してすべてパスすることを確認する**

```
go test ./internal/config/... -v
```

Expected: 4テストすべて PASS。

---

- [ ] **Step 5: コミットする**

```
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: support CLICKUP_API_KEY/TEAM_ID env vars in config.Load"
```

---

## Task 2: `--config` persistent flag を rootCmd に追加する

**Files:**
- Modify: `cmd/clickup/main.go`

---

- [ ] **Step 1: `main.go` を修正する**

`cmd/clickup/main.go` を次の内容に置き換える:

```go
// cmd/clickup/main.go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configPath string

func main() {
	rootCmd := &cobra.Command{
		Use:           "clickup",
		Short:         "ClickUp API CLI wrapper",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path (default: ~/.clickup/config.json)")

	rootCmd.AddCommand(newGetTaskCmd())
	rootCmd.AddCommand(newGetTasksCmd())
	rootCmd.AddCommand(newCreateTaskCmd())
	rootCmd.AddCommand(newUpdateTaskCmd())
	rootCmd.AddCommand(newTimeReportCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

---

- [ ] **Step 2: ビルドして `--help` で flag が表示されることを確認する**

```
go build ./cmd/clickup/... && ./clickup --help
```

Expected: 出力の `Flags:` セクションに `--config string` が含まれる。

---

- [ ] **Step 3: コミットする**

```
git add cmd/clickup/main.go
git commit -m "feat: add --config persistent flag to root command"
```

---

## Task 3: `helpers.go` に `resolveConfigPath()` を追加してパス優先順位チェーンを実装する

**Files:**
- Modify: `cmd/clickup/helpers.go`

---

- [ ] **Step 1: `helpers.go` を修正する**

`cmd/clickup/helpers.go` を次の内容に置き換える:

```go
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
```

---

- [ ] **Step 2: フルビルドとテストを実行する**

```
go build ./... && go test ./...
```

Expected: ビルドエラーなし、全テスト PASS。

---

- [ ] **Step 3: env var だけで `time-report` が動くことを手動確認する**

実際の `CLICKUP_API_KEY` / `CLICKUP_TEAM_ID` を設定して実行する:

```powershell
$env:CLICKUP_API_KEY = "pk_..."
$env:CLICKUP_TEAM_ID = "..."
./clickup time-report --start "2026-05-01T00:00:00+09:00" --end "2026-05-08T00:00:00+09:00"
```

Expected: JSON レポートが stdout に出力される（config file なしで動作）。

---

- [ ] **Step 4: コミットする**

```
git add cmd/clickup/helpers.go
git commit -m "feat: resolve config path via --config flag, CLICKUP_CONFIG env var, or default"
```

---

## Task 4: README.md を更新する

**Files:**
- Modify: `README.md`

---

- [ ] **Step 1: 「セットアップ」セクションに env var と `--config` の説明を追加する**

README.md の `> \`config.json\` はリポジトリ外...` の注記の直後に追記する:

```markdown
### 3. 設定値の上書き

環境変数または `--config` フラグで設定を上書きできる。

**環境変数**

| 環境変数 | 対応フィールド | 説明 |
|---|---|---|
| `CLICKUP_API_KEY` | `apiKey` | config file の値より優先される |
| `CLICKUP_TEAM_ID` | `teamId` | config file の値より優先される |
| `CLICKUP_CONFIG` | 設定ファイルのパス | `--config` フラグより低優先 |

> `CLICKUP_API_KEY` と `CLICKUP_TEAM_ID` が両方設定されていれば、config file がなくても動作する。

**`--config` フラグ**

```bash
clickup --config /path/to/config.json get-tasks
```

すべてのサブコマンドで使用できる。優先順位: `--config` フラグ > `CLICKUP_CONFIG` 環境変数 > `~/.clickup/config.json`

---
```

---

- [ ] **Step 2: コミットする**

```
git add README.md
git commit -m "docs: document env var overrides and --config flag"
```
