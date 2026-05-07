# Timezone Config Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change the default timezone for offset-less datetime strings from JST to UTC, and make the fallback timezone configurable via `AppConfig.Timezone` (IANA name).

**Architecture:** Add `Timezone string` (mapstructure:"timezone") to `AppConfig` with a `TimezoneLocation() *time.Location` method; validate in `Load()`. Change `ParseISO` to accept `*time.Location` as a third argument (nil → UTC). Update all 8 call sites in `cmd/clickup/` to pass `cfg.TimezoneLocation()`.

**Tech Stack:** Go standard library (`time.LoadLocation`), cobra, viper/mapstructure

---

## File Map

| File | Change |
|---|---|
| `internal/dateparse/parse.go` | Add `loc *time.Location` param; replace hardcoded JST with loc (nil→UTC) |
| `internal/dateparse/parse_test.go` | Update `TestParseISO_WithoutOffset` (now UTC); pass loc to all calls |
| `internal/config/config.go` | Add `Timezone string` field; add `TimezoneLocation()` method; validate in `Load()` |
| `cmd/clickup/get_tasks.go` | Pass `cfg.TimezoneLocation()` to both ParseISO calls |
| `cmd/clickup/create_task.go` | Pass `cfg.TimezoneLocation()` to both ParseISO calls; update flag help text |
| `cmd/clickup/update_task.go` | Pass `cfg.TimezoneLocation()` to both ParseISO calls; update flag help text |
| `cmd/clickup/time_report.go` | Pass `cfg.TimezoneLocation()` to both ParseISO calls |
| `config.sample.json` | Add `"timezone": "UTC"` field |
| `README.md` | Update config table; update timezone notes in get-tasks / create-task / update-task |
| `.github/copilot-instructions.md` | Update ParseISO signature note and AppConfig description |

---

## Task 1: Update `ParseISO` — new signature + tests

**Files:**
- Modify: `internal/dateparse/parse.go`
- Modify: `internal/dateparse/parse_test.go`

- [ ] **Step 1: Update the tests first**

Replace the contents of `internal/dateparse/parse_test.go`:

```go
// internal/dateparse/parse_test.go
package dateparse_test

import (
	"testing"
	"time"

	"github.com/hiraking/click-up-cli/internal/dateparse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseISO_WithOffset(t *testing.T) {
	s := "2026-04-19T15:09:41.393+09:00"
	got, err := dateparse.ParseISO(s, "due-after", nil)
	require.NoError(t, err)
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, time.April, got.Month())
	assert.Equal(t, 19, got.Day())
	assert.Equal(t, 15, got.Hour())
	_, offset := got.Zone()
	assert.Equal(t, 9*3600, offset)
}

func TestParseISO_WithZ(t *testing.T) {
	s := "2026-04-19T06:09:41.393Z"
	got, err := dateparse.ParseISO(s, "due-after", nil)
	require.NoError(t, err)
	assert.Equal(t, time.UTC, got.Location())
}

func TestParseISO_WithoutOffset_DefaultsToUTC(t *testing.T) {
	// nil loc → UTC
	s := "2026-04-19T15:09:41"
	got, err := dateparse.ParseISO(s, "due-after", nil)
	require.NoError(t, err)
	assert.Equal(t, time.UTC, got.Location())
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, 15, got.Hour())
}

func TestParseISO_WithoutOffset_UsesProvidedLoc(t *testing.T) {
	// 明示的なタイムゾーンを渡した場合はそれを使う
	jst := time.FixedZone("JST", 9*60*60)
	s := "2026-04-19T15:09:41"
	got, err := dateparse.ParseISO(s, "due-after", jst)
	require.NoError(t, err)
	_, offset := got.Zone()
	assert.Equal(t, 9*3600, offset)
	assert.Equal(t, 15, got.Hour())
}

func TestParseISO_InvalidString(t *testing.T) {
	_, err := dateparse.ParseISO("not-a-date", "due-after", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "due-after")
	assert.Contains(t, err.Error(), "not-a-date")
}
```

- [ ] **Step 2: Run tests — expect compile error (old signature)**

```
go test ./internal/dateparse/...
```

Expected: compile error — `too many arguments in call to dateparse.ParseISO`

- [ ] **Step 3: Update `internal/dateparse/parse.go`**

```go
// internal/dateparse/parse.go
package dateparse

import (
	"fmt"
	"time"
)

// ParseISO は ISO 8601 形式の日時文字列を time.Time に変換する。
// タイムゾーンオフセットが含まれていない場合は loc として解析する。loc が nil のときは time.UTC を使用する。
// optionName はエラーメッセージに使用するオプション名（例: "due-after"）。
func ParseISO(value, optionName string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.UTC
	}

	// オフセット付き / Z 付きのフォーマット群
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, value); err == nil {
			return t, nil
		}
	}

	// オフセットなし → loc として解析
	noOffsetFormats := []string{
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, f := range noOffsetFormats {
		if t, err := time.ParseInLocation(f, value, loc); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf(
		"Error: '--%s' value '%s' is not a valid ISO 8601 datetime.", optionName, value)
}
```

- [ ] **Step 4: Run tests — expect pass**

```
go test ./internal/dateparse/...
```

Expected: `ok  github.com/hiraking/click-up-cli/internal/dateparse`

- [ ] **Step 5: Commit**

```
git add internal/dateparse/parse.go internal/dateparse/parse_test.go
git commit -m "feat(dateparse): add loc param to ParseISO, default UTC

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 2: Update `AppConfig` — `Timezone` field + validation + `TimezoneLocation()`

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Update `internal/config/config.go`**

```go
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// ErrConfigNotFound は設定ファイルが見つからないことを示すセンチネルエラー。
var ErrConfigNotFound = errors.New("config: file not found")

// AppConfig はアプリケーション設定を表す。
type AppConfig struct {
	APIKey   string            `mapstructure:"apiKey"`
	TeamID   string            `mapstructure:"teamId"`
	Lists    map[string]string `mapstructure:"lists"`
	Timezone string            `mapstructure:"timezone"`
}

// TimezoneLocation は Timezone フィールドを *time.Location に変換して返す。
// Timezone が空のときは time.UTC を返す。
// Load() でバリデーション済みのため panic しない。
func (c *AppConfig) TimezoneLocation() *time.Location {
	if c.Timezone == "" {
		return time.UTC
	}
	loc, _ := time.LoadLocation(c.Timezone)
	return loc
}

// Load は設定を読み込む。path が空のときはファイルを読まず env var のみを使う。
// path が指定された場合はファイルを必須とする。
// CLICKUP_API_KEY / CLICKUP_TEAM_ID 環境変数はファイルの値を上書きする。
// apiKey または teamId が空の場合はバリデーションエラーを返す。
// timezone が空でない場合、有効な IANA タイムゾーン名かどうかを検証する。
func Load(path string) (*AppConfig, error) {
	v := viper.New()

	if path != "" {
		v.SetConfigFile(path)

		if err := v.ReadInConfig(); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("config file not found: %w", ErrConfigNotFound)
			}
			var pathErr *os.PathError
			if errors.As(err, &pathErr) && errors.Is(pathErr.Err, os.ErrNotExist) {
				return nil, fmt.Errorf("config file not found: %w", ErrConfigNotFound)
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
	if cfg.Timezone != "" {
		if _, err := time.LoadLocation(cfg.Timezone); err != nil {
			return nil, fmt.Errorf("config: invalid timezone %q: %w", cfg.Timezone, err)
		}
	}

	return &cfg, nil
}
```

- [ ] **Step 2: Verify it builds**

```
go build ./...
```

Expected: no errors (call sites not yet updated — they'll fail because ParseISO now requires 3 args)

Actually at this point the call sites still use the old 2-arg ParseISO, so build will fail. That's fine — proceed to Task 3 immediately.

- [ ] **Step 3: Commit (after Task 3 is done)**

Hold this commit until call sites are updated. Skip for now — see Task 3 Step 6.

---

## Task 3: Update all call sites in `cmd/clickup/`

**Files:**
- Modify: `cmd/clickup/get_tasks.go`
- Modify: `cmd/clickup/create_task.go`
- Modify: `cmd/clickup/update_task.go`
- Modify: `cmd/clickup/time_report.go`

- [ ] **Step 1: Update `cmd/clickup/get_tasks.go`**

Change both ParseISO calls (lines 47, 54) to pass `cfg.TimezoneLocation()`:

```go
if dueAfterStr != "" {
    t, err := dateparse.ParseISO(dueAfterStr, "due-after", cfg.TimezoneLocation())
    if err != nil {
        return err
    }
    dueDateGt = &t
}
if dueBeforeStr != "" {
    t, err := dateparse.ParseISO(dueBeforeStr, "due-before", cfg.TimezoneLocation())
    if err != nil {
        return err
    }
    dueDateLt = &t
}
```

- [ ] **Step 2: Update `cmd/clickup/create_task.go`**

Change both ParseISO calls (lines 67, 74) and update flag help texts:

```go
if cmd.Flags().Changed("due-date") {
    t, err := dateparse.ParseISO(dueDateStr, "due-date", cfg.TimezoneLocation())
    if err != nil {
        return err
    }
    req.DueDate = &t
}
if cmd.Flags().Changed("start-date") {
    t, err := dateparse.ParseISO(startDateStr, "start-date", cfg.TimezoneLocation())
    if err != nil {
        return err
    }
    req.StartDate = &t
}
```

Flag help texts (replace `treated as JST (+09:00)` → `treated as UTC or the timezone set in config`):

```go
cmd.Flags().StringVar(&dueDateStr, "due-date", "", "Due date as ISO 8601. Timezone-less values use the timezone from config (default UTC).")
cmd.Flags().StringVar(&startDateStr, "start-date", "", "Start date as ISO 8601. Timezone-less values use the timezone from config (default UTC).")
```

- [ ] **Step 3: Update `cmd/clickup/update_task.go`**

Change both ParseISO calls (lines 97, 104):

```go
if changed("due-date") {
    t, err := dateparse.ParseISO(dueDateStr, "due-date", cfg.TimezoneLocation())
    if err != nil {
        return err
    }
    req.DueDate = &t
}
if changed("start-date") {
    t, err := dateparse.ParseISO(startDateStr, "start-date", cfg.TimezoneLocation())
    if err != nil {
        return err
    }
    req.StartDate = &t
}
```

Flag help texts:

```go
cmd.Flags().StringVar(&dueDateStr, "due-date", "", "New due date as ISO 8601. Timezone-less values use the timezone from config (default UTC).")
cmd.Flags().StringVar(&startDateStr, "start-date", "", "New start date as ISO 8601. Timezone-less values use the timezone from config (default UTC).")
```

- [ ] **Step 4: Update `cmd/clickup/time_report.go`**

Change both ParseISO calls (lines 31, 35):

```go
start, err := dateparse.ParseISO(flagStart, "start", cfg.TimezoneLocation())
if err != nil {
    return err
}
end, err := dateparse.ParseISO(flagEnd, "end", cfg.TimezoneLocation())
if err != nil {
    return err
}
```

- [ ] **Step 5: Build to verify everything compiles**

```
go build ./...
```

Expected: no errors

- [ ] **Step 6: Run all tests**

```
go test ./...
```

Expected: all pass

- [ ] **Step 7: Commit**

```
git add internal/config/config.go cmd/clickup/get_tasks.go cmd/clickup/create_task.go cmd/clickup/update_task.go cmd/clickup/time_report.go
git commit -m "feat(config): add timezone field; update ParseISO call sites

- AppConfig gains Timezone string (IANA name, empty = UTC)
- TimezoneLocation() helper method on AppConfig
- All ParseISO call sites now pass cfg.TimezoneLocation()

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 4: Update docs and sample config

**Files:**
- Modify: `config.sample.json`
- Modify: `README.md`
- Modify: `.github/copilot-instructions.md`

- [ ] **Step 1: Update `config.sample.json`**

```json
{
  "apiKey": "pk_YOUR_API_KEY_HERE",
  "teamId": "YOUR_TEAM_ID_HERE",
  "lists": {
    "my-list": "LIST_ID_HERE"
  },
  "timezone": "UTC"
}
```

- [ ] **Step 2: Update `README.md` — config table**

In the "Create a config file" section, update the sample JSON and config table:

Sample JSON becomes:
```json
{
  "apiKey": "pk_YOUR_API_KEY_HERE",
  "teamId": "YOUR_TEAM_ID_HERE",
  "lists": {
    "work":  "LIST_ID_1",
    "study": "LIST_ID_2"
  },
  "timezone": "UTC"
}
```

Add a row to the config field table:

```markdown
| `timezone` | IANA timezone name for offset-less datetime strings (e.g. `"Asia/Tokyo"`, `"UTC"`). Defaults to `"UTC"` if omitted. |
```

- [ ] **Step 3: Update `README.md` — timezone notes in get-tasks**

Find the note:
```
> - **Timezone:** Datetime strings without an offset (e.g. `"2026-05-01"`, `"2026-05-01T09:00"`) are interpreted as JST (+09:00). Explicit offsets (e.g. `"2026-05-01T00:00:00Z"`) are used as-is.
```

Replace with:
```
> - **Timezone:** Datetime strings without an offset (e.g. `"2026-05-01"`, `"2026-05-01T09:00"`) are interpreted using the `timezone` setting from config (default: UTC). Explicit offsets (e.g. `"2026-05-01T00:00:00Z"`) are used as-is.
```

- [ ] **Step 4: Update `.github/copilot-instructions.md`**

Find the `ParseISO` description:
```
`ParseISO(value, optionName string) (time.Time, error)` — 2 引数シグネチャ必須。
オフセットなし文字列は JST (`Asia/Tokyo`) にフォールバックする。
```

Replace with:
```
`ParseISO(value, optionName string, loc *time.Location) (time.Time, error)` — 3 引数シグネチャ。
`loc` が `nil` のときは `time.UTC` を使用する。
オフセットなし文字列は `loc` にフォールバックする。
```

Also find the `AppConfig` description and add `Timezone string` field:
```
`AppConfig` / `Load` (`internal/config/config.go`)
```
Update the field description to include:
```
`Timezone` / `timezone` フィールド（IANA タイムゾーン名、省略時は UTC）。
`TimezoneLocation() *time.Location` メソッドで `*time.Location` に変換する。
```

- [ ] **Step 5: Commit**

```
git add config.sample.json README.md .github/copilot-instructions.md
git commit -m "docs: update timezone default to UTC, add timezone config docs

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```
