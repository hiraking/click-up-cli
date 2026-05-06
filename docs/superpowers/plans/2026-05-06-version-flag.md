# --version フラグの追加 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `clickup --version` でバージョン番号を表示できるようにする。

**Architecture:** cobra の組み込み `Version` フィールドに `var version = "dev"` を設定し、リリースビルド時に `-ldflags "-X main.version=$Version"` で上書きする。あわせて `-trimpath -ldflags "-s -w"` でバイナリサイズを削減する。

**Tech Stack:** Go, cobra v1.10.2, PowerShell (release.ps1)

---

### Task 1: main.go に version 変数と cobra Version フィールドを追加する

**Files:**
- Modify: `cmd/clickup/main.go`

- [ ] **Step 1: main.go を編集する**

`var configPath string` の直前に `var version = "dev"` を追加し、`rootCmd` の初期化に `Version: version,` を追加する。

```go
// cmd/clickup/main.go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configPath string
var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:           "clickup",
		Short:         "ClickUp API CLI wrapper",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path (default: ~/.clickup/config.json)")

	rootCmd.AddCommand(newGetTaskCmd())
	rootCmd.AddCommand(newGetTasksCmd())
	rootCmd.AddCommand(newCreateTaskCmd())
	rootCmd.AddCommand(newUpdateTaskCmd())
	rootCmd.AddCommand(newTimeReportCmd())
	rootCmd.AddCommand(newShowConfigCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: ビルドして動作確認する**

```powershell
go build -o dist/clickup-dev.exe ./cmd/clickup
.\dist\clickup-dev.exe --version
```

期待する出力:
```
clickup version dev
```

- [ ] **Step 3: ldflags でバージョンを埋め込んで確認する**

```powershell
go build -trimpath -ldflags "-s -w -X main.version=v0.3.0" -o dist/clickup-test.exe ./cmd/clickup
.\dist\clickup-test.exe --version
```

期待する出力:
```
clickup version v0.3.0
```

- [ ] **Step 4: dist/ の一時ファイルを削除してコミットする**

```powershell
Remove-Item dist\* -Force -ErrorAction SilentlyContinue
git add cmd/clickup/main.go
git commit -m "feat: add --version flag using cobra Version field"
```

---

### Task 2: release.ps1 のビルドフラグを更新する

**Files:**
- Modify: `.github/skills/create-github-release/release.ps1:38`

- [ ] **Step 1: release.ps1 の go build 行を編集する**

38行目の `go build -o $t.Out $pkg` を以下に変更する:

```powershell
    go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $t.Out $pkg
```

変更後のループ全体:

```powershell
foreach ($t in $targets) {
    Write-Host "==> Building $($t.Out)"
    $env:GOOS   = $t.OS
    $env:GOARCH = $t.Arch
    go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $t.Out $pkg
}
```

- [ ] **Step 2: コミットする**

```powershell
git add .github/skills/create-github-release/release.ps1
git commit -m "build: add trimpath and ldflags to release build for version embedding and size reduction"
```
