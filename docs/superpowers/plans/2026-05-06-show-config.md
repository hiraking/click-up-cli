# show-config Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `show-config` コマンドを追加し、APIキーをマスクした上で現在の設定をJSON形式で標準出力に表示する。

**Architecture:** `cmd/clickup/helpers.go` に `maskAPIKey` ヘルパーを追加し、`cmd/clickup/show_config.go` に新コマンドを定義。`main.go` でコマンドを登録する。出力用の無名構造体を使い `printJSON` で既存フォーマットに揃える。

**Tech Stack:** Go, cobra, testify

---

## ファイル構成

| 操作 | ファイル |
|------|---------|
| 新規作成 | `cmd/clickup/show_config.go` |
| 新規作成 | `cmd/clickup/show_config_test.go` |
| 変更 | `cmd/clickup/helpers.go`（`maskAPIKey` 追加） |
| 変更 | `cmd/clickup/main.go`（コマンド登録） |

---

### Task 1: `maskAPIKey` ヘルパーをテスト駆動で追加

**Files:**
- Modify: `cmd/clickup/helpers.go`
- Create: `cmd/clickup/show_config_test.go`

- [ ] **Step 1: テストを書く**

`cmd/clickup/show_config_test.go` を新規作成：

```go
// cmd/clickup/show_config_test.go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"pk_abcdefgh1234", "****1234"},
		{"12345", "****2345"},
		{"abcd", "****"},
		{"abc", "****"},
		{"", "****"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, maskAPIKey(tt.input))
		})
	}
}
```

- [ ] **Step 2: テストが失敗することを確認する**

```
go test ./cmd/clickup/... -run TestMaskAPIKey -v
```

期待結果: `FAIL` — `maskAPIKey` が未定義

- [ ] **Step 3: `maskAPIKey` を `helpers.go` に追加する**

`cmd/clickup/helpers.go` の末尾に追記：

```go
func maskAPIKey(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}
```

- [ ] **Step 4: テストが通ることを確認する**

```
go test ./cmd/clickup/... -run TestMaskAPIKey -v
```

期待結果: 5件すべて `PASS`

- [ ] **Step 5: コミット**

```
git add cmd/clickup/helpers.go cmd/clickup/show_config_test.go
git commit -m "feat: add maskAPIKey helper"
```

---

### Task 2: `show-config` コマンドを実装する

**Files:**
- Create: `cmd/clickup/show_config.go`

- [ ] **Step 1: `show_config.go` を作成する**

```go
// cmd/clickup/show_config.go
package main

import (
	"github.com/spf13/cobra"
)

func newShowConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-config",
		Short: "Show current configuration as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			out := struct {
				APIKey string            `json:"apiKey"`
				TeamID string            `json:"teamId"`
				Lists  map[string]string `json:"lists"`
			}{
				APIKey: maskAPIKey(cfg.APIKey),
				TeamID: cfg.TeamID,
				Lists:  cfg.Lists,
			}
			return printJSON(out)
		},
	}
	return cmd
}
```

- [ ] **Step 2: ビルドが通ることを確認する**

```
go build ./cmd/clickup/...
```

期待結果: エラーなし

- [ ] **Step 3: コミット**

```
git add cmd/clickup/show_config.go
git commit -m "feat: add show-config command"
```

---

### Task 3: `main.go` にコマンドを登録する

**Files:**
- Modify: `cmd/clickup/main.go`

- [ ] **Step 1: `newShowConfigCmd()` を `main.go` に追加する**

`cmd/clickup/main.go` の `rootCmd.AddCommand(newTimeReportCmd())` の次の行に追加：

```go
rootCmd.AddCommand(newShowConfigCmd())
```

変更後の `main.go` の該当箇所：

```go
	rootCmd.AddCommand(newGetTaskCmd())
	rootCmd.AddCommand(newGetTasksCmd())
	rootCmd.AddCommand(newCreateTaskCmd())
	rootCmd.AddCommand(newUpdateTaskCmd())
	rootCmd.AddCommand(newTimeReportCmd())
	rootCmd.AddCommand(newShowConfigCmd())
```

- [ ] **Step 2: 全テストが通ることを確認する**

```
go test ./...
```

期待結果: すべて `PASS`

- [ ] **Step 3: 手動動作確認**

```
go run ./cmd/clickup show-config
```

期待結果（例）:

```json
{
  "apiKey": "****a1b2",
  "teamId": "12345678",
  "lists": {
    "my-list": "87654321"
  }
}
```

- [ ] **Step 4: コミット**

```
git add cmd/clickup/main.go
git commit -m "feat: register show-config command"
```
