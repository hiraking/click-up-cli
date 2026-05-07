# task-type Config Mapping Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace hardcoded `--task-type` aliases with a `taskTypes` map in `config.json`, making the flag portable across any ClickUp workspace.

**Architecture:** Remove the `TaskType` int-enum type and named constants from the model layer; change `CustomItemID` to `*int`. Add an optional `TaskTypes map[string]int` field to `AppConfig`. Resolve `--task-type` at runtime via config lookup inside the `create-task` command handler.

**Tech Stack:** Go 1.22+, cobra, viper, testify

---

## File Map

| File | Change |
|---|---|
| `internal/models/create_task.go` | Remove `TaskType` type + constants; `CustomItemID *TaskType` → `*int` |
| `internal/client/mapper.go` | Remove redundant `int()` cast for `CustomItemID` |
| `internal/client/mapper_test.go` | Use raw `int` literal instead of `models.TaskTypeMilestone` |
| `internal/config/config.go` | Add `TaskTypes map[string]int` field |
| `internal/config/config_test.go` | Add test for `taskTypes` loading |
| `cmd/clickup/create_task.go` | Remove `parseTaskType()`; add `lookupTaskType()` + `sortedStringKeys()`; update handler |
| `cmd/clickup/create_task_test.go` | Remove `TestParseTaskType_*`; add `TestLookupTaskType_*` |
| `config.sample.json` | Add `taskTypes` example |
| `README.md` | Update `--task-type` docs + config table |

---

### Task 1: Update model and mapper — `CustomItemID *int`

**Files:**
- Modify: `internal/models/create_task.go`
- Modify: `internal/client/mapper.go`
- Modify: `internal/client/mapper_test.go`

- [ ] **Step 1: Update the mapper test to use a raw `int` literal**

  Open `internal/client/mapper_test.go` and replace the `TestMapToRawCreateBody_CustomItemID` function:

  ```go
  func TestMapToRawCreateBody_CustomItemID(t *testing.T) {
  	id := 1
  	req := models.CreateTaskRequest{
  		Name:         "Milestone Task",
  		CustomItemID: &id,
  	}

  	body := mapToRawCreateBody(req)

  	require.NotNil(t, body.CustomItemID)
  	assert.Equal(t, 1, *body.CustomItemID)
  }
  ```

- [ ] **Step 2: Run the test — verify it fails to compile**

  ```
  cd C:\Users\平木大都\source\repos\playground\mine\click-up-cli
  go test ./internal/client/...
  ```

  Expected: compile error — `models.TaskTypeMilestone undefined` (or similar type mismatch once we start editing).

- [ ] **Step 3: Remove `TaskType` from the model and change `CustomItemID` to `*int`**

  Replace the entire `internal/models/create_task.go` with:

  ```go
  // internal/models/create_task.go
  package models

  import "time"

  // TaskPriority は ClickUp の優先度を表す。API には int として送信する。
  type TaskPriority int

  const (
  	PriorityUrgent TaskPriority = 1
  	PriorityHigh   TaskPriority = 2
  	PriorityNormal TaskPriority = 3
  	PriorityLow    TaskPriority = 4
  )

  // CreateTaskRequest はタスク作成リクエストのパラメータ。
  type CreateTaskRequest struct {
  	Name         string
  	ParentID     *string
  	Description  *string
  	Status       *string
  	Priority     *TaskPriority
  	DueDate      *time.Time
  	StartDate    *time.Time
  	TimeEstimate *time.Duration // 分単位で渡し、API には ms として送信
  	CustomItemID *int
  }

  // UpdateTaskRequest はタスク更新リクエストのパラメータ。
  // nil フィールドは更新しない。ClearFields に含まれるフィールドは値をクリアする。
  type UpdateTaskRequest struct {
  	Name         *string
  	Description  *string
  	Status       *string
  	Priority     *TaskPriority
  	DueDate      *time.Time
  	StartDate    *time.Time
  	TimeEstimate *time.Duration
  	Parent       *string
  	ClearFields  []string
  }
  ```

- [ ] **Step 4: Simplify the mapper — remove the `int()` cast for `CustomItemID`**

  In `internal/client/mapper.go`, replace:

  ```go
  if req.CustomItemID != nil {
  	id := int(*req.CustomItemID)
  	body.CustomItemID = &id
  }
  ```

  with:

  ```go
  if req.CustomItemID != nil {
  	body.CustomItemID = req.CustomItemID
  }
  ```

- [ ] **Step 5: Run all tests — verify they pass**

  ```
  go test ./...
  ```

  Expected: all tests pass (the `TestParseTaskType_*` tests in `cmd/clickup` will fail to compile because `parseTaskType` still references `models.TaskType` — fix those by temporarily deleting or commenting them out before moving on, OR proceed directly to Task 3 which replaces the tests properly).

  > If `cmd/clickup/create_task_test.go` fails to compile because of removed `models.TaskType`, comment out the two `TestParseTaskType_*` functions for now. They will be replaced in Task 3.

- [ ] **Step 6: Commit**

  ```bash
  git add internal/models/create_task.go internal/client/mapper.go internal/client/mapper_test.go cmd/clickup/create_task_test.go
  git commit -m "refactor: replace TaskType enum with plain int for CustomItemID

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
  ```

---

### Task 2: Add `TaskTypes` to config

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write a failing test for `taskTypes` loading**

  Add the following test to `internal/config/config_test.go`:

  ```go
  func TestLoad_TaskTypes(t *testing.T) {
  	dir := t.TempDir()
  	path := filepath.Join(dir, "config.json")
  	require.NoError(t, os.WriteFile(path, []byte(`{
  		"apiKey": "pk_key",
  		"teamId": "team",
  		"taskTypes": { "milestone": 1, "project": 1001 }
  	}`), 0600))

  	t.Setenv("CLICKUP_API_KEY", "")
  	t.Setenv("CLICKUP_TEAM_ID", "")

  	cfg, err := config.Load(path)
  	require.NoError(t, err)
  	assert.Equal(t, map[string]int{"milestone": 1, "project": 1001}, cfg.TaskTypes)
  }

  func TestLoad_TaskTypes_Absent(t *testing.T) {
  	dir := t.TempDir()
  	path := filepath.Join(dir, "config.json")
  	require.NoError(t, os.WriteFile(path, []byte(`{"apiKey":"pk_key","teamId":"team"}`), 0600))

  	t.Setenv("CLICKUP_API_KEY", "")
  	t.Setenv("CLICKUP_TEAM_ID", "")

  	cfg, err := config.Load(path)
  	require.NoError(t, err)
  	assert.Empty(t, cfg.TaskTypes)
  }
  ```

- [ ] **Step 2: Run the tests — verify they fail**

  ```
  go test ./internal/config/...
  ```

  Expected: FAIL — `cfg.TaskTypes` field does not exist yet.

- [ ] **Step 3: Add `TaskTypes` field to `AppConfig`**

  In `internal/config/config.go`, update the `AppConfig` struct:

  ```go
  // AppConfig はアプリケーション設定を表す。
  type AppConfig struct {
  	APIKey    string            `mapstructure:"apiKey"`
  	TeamID    string            `mapstructure:"teamId"`
  	Lists     map[string]string `mapstructure:"lists"`
  	Timezone  string            `mapstructure:"timezone"`
  	TaskTypes map[string]int    `mapstructure:"taskTypes"`
  }
  ```

  No other changes — the field is optional and requires no validation.

- [ ] **Step 4: Run the tests — verify they pass**

  ```
  go test ./internal/config/...
  ```

  Expected: all pass.

- [ ] **Step 5: Commit**

  ```bash
  git add internal/config/config.go internal/config/config_test.go
  git commit -m "feat: add TaskTypes field to AppConfig

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
  ```

---

### Task 3: Replace `parseTaskType` with `lookupTaskType` and config-based handler

**Files:**
- Modify: `cmd/clickup/create_task.go`
- Modify: `cmd/clickup/create_task_test.go`

- [ ] **Step 1: Write failing tests for `lookupTaskType`**

  Replace the entire contents of `cmd/clickup/create_task_test.go` with:

  ```go
  // cmd/clickup/create_task_test.go
  package main

  import (
  	"testing"

  	"github.com/stretchr/testify/assert"
  	"github.com/stretchr/testify/require"
  )

  func TestLookupTaskType_ValidKey(t *testing.T) {
  	taskTypes := map[string]int{"milestone": 1, "project": 1001}

  	id, err := lookupTaskType(taskTypes, "milestone")
  	require.NoError(t, err)
  	assert.Equal(t, 1, id)
  }

  func TestLookupTaskType_UnknownKey(t *testing.T) {
  	taskTypes := map[string]int{"milestone": 1, "project": 1001}

  	_, err := lookupTaskType(taskTypes, "foo")
  	require.Error(t, err)
  	assert.Contains(t, err.Error(), "Unknown task type 'foo'")
  	assert.Contains(t, err.Error(), "milestone")
  	assert.Contains(t, err.Error(), "project")
  }

  func TestLookupTaskType_EmptyConfig(t *testing.T) {
  	_, err := lookupTaskType(nil, "milestone")
  	require.Error(t, err)
  	assert.Contains(t, err.Error(), "No task types configured")
  }

  func TestLookupTaskType_KeysSorted(t *testing.T) {
  	taskTypes := map[string]int{"zebra": 3, "alpha": 1, "milestone": 2}

  	_, err := lookupTaskType(taskTypes, "unknown")
  	require.Error(t, err)
  	// Available list should be alphabetically sorted
  	assert.Contains(t, err.Error(), "alpha, milestone, zebra")
  }
  ```

- [ ] **Step 2: Run the tests — verify they fail**

  ```
  go test ./cmd/clickup/...
  ```

  Expected: FAIL — `lookupTaskType` is undefined.

- [ ] **Step 3: Implement `lookupTaskType` and `sortedStringKeys`, remove `parseTaskType`**

  Replace the entire `cmd/clickup/create_task.go` with:

  ```go
  // cmd/clickup/create_task.go
  package main

  import (
  	"context"
  	"fmt"
  	"sort"
  	"strings"
  	"time"

  	"github.com/hiraking/click-up-cli/internal/client"
  	"github.com/hiraking/click-up-cli/internal/dateparse"
  	"github.com/hiraking/click-up-cli/internal/models"
  	"github.com/spf13/cobra"
  )

  func newCreateTaskCmd() *cobra.Command {
  	var listName string
  	var description string
  	var parentID string
  	var status string
  	var priority string
  	var dueDateStr string
  	var startDateStr string
  	var timeEstimateMin int
  	var taskTypeStr string

  	cmd := &cobra.Command{
  		Use:   "create-task <name>",
  		Short: "Create a new task and output it as JSON",
  		Args:  cobra.ExactArgs(1),
  		RunE: func(cmd *cobra.Command, args []string) error {
  			name := args[0]

  			cfg, err := loadConfig()
  			if err != nil {
  				return err
  			}

  			listID, ok := cfg.Lists[listName]
  			if !ok {
  				return fmt.Errorf("Error: Unknown list name '%s'. Available: %s",
  					listName, availableListNames(cfg.Lists))
  			}

  			req := models.CreateTaskRequest{Name: name}

  			if cmd.Flags().Changed("description") {
  				req.Description = &description
  			}
  			if cmd.Flags().Changed("parent") {
  				if strings.TrimSpace(parentID) == "" {
  					return fmt.Errorf("Error: '--parent' must not be empty or whitespace.")
  				}
  				req.ParentID = &parentID
  			}
  			if cmd.Flags().Changed("status") {
  				req.Status = &status
  			}
  			if cmd.Flags().Changed("priority") {
  				p, err := parsePriority(priority)
  				if err != nil {
  					return err
  				}
  				req.Priority = &p
  			}
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
  			if cmd.Flags().Changed("time-estimate") {
  				d := time.Duration(timeEstimateMin) * time.Minute
  				req.TimeEstimate = &d
  			}
  			if cmd.Flags().Changed("task-type") {
  				id, err := lookupTaskType(cfg.TaskTypes, taskTypeStr)
  				if err != nil {
  					return err
  				}
  				req.CustomItemID = &id
  			}

  			c := client.New(cfg.APIKey)
  			task, err := c.CreateTask(context.Background(), listID, req)
  			if err != nil {
  				return err
  			}
  			return printJSON(task)
  		},
  	}

  	cmd.Flags().StringVar(&listName, "list", "", "List name defined in config.json.")
  	_ = cmd.MarkFlagRequired("list")
  	cmd.Flags().StringVar(&description, "description", "", "Task description.")
  	cmd.Flags().StringVar(&parentID, "parent", "", "Parent task ID. Creates a subtask.")
  	cmd.Flags().StringVar(&status, "status", "", "Status name (e.g. \"to do\", \"in progress\").")
  	cmd.Flags().StringVar(&priority, "priority", "", "Priority: urgent, high, normal, or low.")
  	cmd.Flags().StringVar(&dueDateStr, "due-date", "", "Due date as ISO 8601. Timezone-less values use the timezone from config (default UTC).")
  	cmd.Flags().StringVar(&startDateStr, "start-date", "", "Start date as ISO 8601. Timezone-less values use the timezone from config (default UTC).")
  	cmd.Flags().IntVar(&timeEstimateMin, "time-estimate", 0, "Time estimate in minutes.")
  	cmd.Flags().StringVar(&taskTypeStr, "task-type", "", "Task type name as defined in the taskTypes config.")

  	return cmd
  }

  func parsePriority(s string) (models.TaskPriority, error) {
  	switch strings.ToLower(s) {
  	case "urgent":
  		return models.PriorityUrgent, nil
  	case "high":
  		return models.PriorityHigh, nil
  	case "normal":
  		return models.PriorityNormal, nil
  	case "low":
  		return models.PriorityLow, nil
  	default:
  		return 0, fmt.Errorf("Error: Invalid priority '%s'. Use urgent, high, normal, or low.", s)
  	}
  }

  // lookupTaskType は cfg.TaskTypes から name に対応する custom_item_id を返す。
  // taskTypes が空の場合、または name が見つからない場合はエラーを返す。
  func lookupTaskType(taskTypes map[string]int, name string) (int, error) {
  	if len(taskTypes) == 0 {
  		return 0, fmt.Errorf("Error: No task types configured. Add a \"taskTypes\" mapping to config.json.")
  	}
  	id, ok := taskTypes[name]
  	if !ok {
  		keys := sortedStringKeys(taskTypes)
  		return 0, fmt.Errorf("Error: Unknown task type '%s'. Available: %s", name, strings.Join(keys, ", "))
  	}
  	return id, nil
  }

  func sortedStringKeys(m map[string]int) []string {
  	keys := make([]string, 0, len(m))
  	for k := range m {
  		keys = append(keys, k)
  	}
  	sort.Strings(keys)
  	return keys
  }
  ```

- [ ] **Step 4: Run the tests — verify they pass**

  ```
  go test ./cmd/clickup/...
  ```

  Expected: all pass.

- [ ] **Step 5: Run all tests to ensure nothing is broken**

  ```
  go test ./...
  ```

  Expected: all pass.

- [ ] **Step 6: Commit**

  ```bash
  git add cmd/clickup/create_task.go cmd/clickup/create_task_test.go
  git commit -m "feat: resolve --task-type from config.json taskTypes mapping

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
  ```

---

### Task 4: Update `config.sample.json` and `README.md`

**Files:**
- Modify: `config.sample.json`
- Modify: `README.md`

- [ ] **Step 1: Add `taskTypes` to `config.sample.json`**

  Replace the entire file with:

  ```json
  {
    "apiKey": "pk_YOUR_API_KEY_HERE",
    "teamId": "YOUR_TEAM_ID_HERE",
    "lists": {
      "my-list": "LIST_ID_HERE"
    },
    "timezone": "UTC",
    "taskTypes": {
      "milestone": 1
    }
  }
  ```

- [ ] **Step 2: Update the config table in `README.md` Setup section**

  In the config field table (around line 37), add a row for `taskTypes`:

  | Field | Description |
  |---|---|
  | `apiKey` | Personal API Token (Settings → Apps → API Token) |
  | `teamId` | Workspace ID (found in the URL: `/w/{teamId}/`) |
  | `lists` | Name-to-ID mapping used by the `--list` flag |
  | `timezone` | IANA timezone name for offset-less datetime strings (e.g. `"Asia/Tokyo"`, `"UTC"`). Defaults to `"UTC"` if omitted. |
  | `taskTypes` | Optional. Name-to-ID mapping for `--task-type`. Keys are arbitrary strings; values are `custom_item_id` integers from your ClickUp workspace. |

- [ ] **Step 3: Update the `--task-type` row in the `create-task` options table**

  Change the `--task-type` row (around line 145) from:

  ```
  | `--task-type <name>` | string | `milestone` / `project` / `book` |
  ```

  to:

  ```
  | `--task-type <name>` | string | Task type name as defined in `taskTypes` config. |
  ```

- [ ] **Step 4: Update the example in Error handling section**

  In the Error handling table (around line 356), add or update the `--task-type` error rows:

  | Case | Example message |
  |---|---|
  | `taskTypes` not configured | `Error: No task types configured. Add a "taskTypes" mapping to config.json.` |
  | Unknown task type | `Error: Unknown task type 'foo'. Available: milestone, project` |

- [ ] **Step 5: Run all tests as final verification**

  ```
  go test ./...
  ```

  Expected: all pass.

- [ ] **Step 6: Commit**

  ```bash
  git add config.sample.json README.md
  git commit -m "docs: update config.sample.json and README for taskTypes mapping

  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
  ```
