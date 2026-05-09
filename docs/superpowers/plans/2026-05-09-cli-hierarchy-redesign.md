# CLI Hierarchy Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure flat CLI commands (`get-tasks`, `create-task`, etc.) into a resource-oriented hierarchy (`task list`, `task get`, `time report`, `config show`).

**Architecture:** Each resource group becomes its own Go package under `cmd/clickup/<resource>/` exporting `NewCmd(configPath *string) *cobra.Command`. Shared CLI utilities (JSON output, config loading, formatting) move to a new `cmdutil` package. `main.go` wires the three resource commands up to the root command via a pointer to the `configPath` variable.

**Tech Stack:** Go, github.com/spf13/cobra, github.com/stretchr/testify

---

## File Map

**Create:**
- `cmd/clickup/cmdutil/config.go` — `LoadConfig`, `ResolveConfigPath`
- `cmd/clickup/cmdutil/output.go` — `PrintJSON`
- `cmd/clickup/cmdutil/format.go` — `AvailableListNames`, `MaskAPIKey`
- `cmd/clickup/cmdutil/format_test.go` — `TestMaskAPIKey`
- `cmd/clickup/task/cmd.go` — `NewCmd(configPath *string)`
- `cmd/clickup/task/list.go` — `newListCmd` (was `get-tasks`)
- `cmd/clickup/task/get.go` — `newGetCmd` (was `get-task`)
- `cmd/clickup/task/create.go` — `newCreateCmd`, `parsePriority`, `lookupTaskType`, `sortedStringKeys` (was `create-task`)
- `cmd/clickup/task/update.go` — `newUpdateCmd`, `validClearFields` (was `update-task`)
- `cmd/clickup/task/delete.go` — `newDeleteCmd`, `deleteTaskResult` (was `delete-task`)
- `cmd/clickup/task/create_test.go` — `TestLookupTaskType_*`
- `cmd/clickup/task/update_test.go` — `TestUpdateCmd_*`
- `cmd/clickup/time/cmd.go` — `NewCmd(configPath *string)` (package `timecmd`)
- `cmd/clickup/time/report.go` — `newReportCmd` (was `time-report`)
- `cmd/clickup/config/cmd.go` — `NewCmd(configPath *string)` (package `configcmd`)
- `cmd/clickup/config/show.go` — `newShowCmd` (was `show-config`)

**Modify:**
- `cmd/clickup/main.go` — rewrite: remove old AddCommand calls, wire new packages

**Delete:**
- `cmd/clickup/get_tasks.go`
- `cmd/clickup/get_task.go`
- `cmd/clickup/create_task.go`
- `cmd/clickup/create_task_test.go`
- `cmd/clickup/update_task.go`
- `cmd/clickup/update_task_test.go`
- `cmd/clickup/delete_task.go`
- `cmd/clickup/time_report.go`
- `cmd/clickup/show_config.go`
- `cmd/clickup/helpers.go`
- `cmd/clickup/helpers_test.go`

**Update:**
- `README.md` — rewrite `## Commands` section with new command names

---

## Task 1: Create cmdutil package

**Files:**
- Create: `cmd/clickup/cmdutil/format_test.go`
- Create: `cmd/clickup/cmdutil/format.go`
- Create: `cmd/clickup/cmdutil/output.go`
- Create: `cmd/clickup/cmdutil/config.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/clickup/cmdutil/format_test.go`:

```go
// cmd/clickup/cmdutil/format_test.go
package cmdutil

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
			assert.Equal(t, tt.expected, MaskAPIKey(tt.input))
		})
	}
}
```

- [ ] **Step 2: Run test to confirm it fails (compile error)**

```
go test ./cmd/clickup/cmdutil/...
```

Expected: compile error — `undefined: MaskAPIKey`

- [ ] **Step 3: Create cmdutil/format.go**

```go
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
```

- [ ] **Step 4: Create cmdutil/output.go**

```go
// cmd/clickup/cmdutil/output.go
package cmdutil

import (
	"encoding/json"
	"os"
)

func PrintJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
```

- [ ] **Step 5: Create cmdutil/config.go**

```go
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
```

- [ ] **Step 6: Run tests to confirm they pass**

```
go test ./cmd/clickup/cmdutil/...
```

Expected: `ok  github.com/hiraking/click-up-cli/cmd/clickup/cmdutil`

- [ ] **Step 7: Commit**

```
git add cmd/clickup/cmdutil/
git commit -m "feat: add cmdutil package with shared CLI utilities"
```

---

## Task 2: Create task package

**Files:**
- Create: `cmd/clickup/task/create_test.go`
- Create: `cmd/clickup/task/update_test.go`
- Create: `cmd/clickup/task/cmd.go`
- Create: `cmd/clickup/task/list.go`
- Create: `cmd/clickup/task/get.go`
- Create: `cmd/clickup/task/create.go`
- Create: `cmd/clickup/task/update.go`
- Create: `cmd/clickup/task/delete.go`

- [ ] **Step 1: Write the failing tests**

Create `cmd/clickup/task/create_test.go`:

```go
// cmd/clickup/task/create_test.go
package task

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
	assert.Contains(t, err.Error(), "alpha, milestone, zebra")
}
```

Create `cmd/clickup/task/update_test.go`:

```go
// cmd/clickup/task/update_test.go
package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCmd_ArchiveAndUnarchiveTogether(t *testing.T) {
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123", "--archive", "--unarchive"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be used together")
}

func TestUpdateCmd_NoFlagsProvided(t *testing.T) {
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields specified")
}

func TestUpdateCmd_ArchiveFlag_ValidationPasses(t *testing.T) {
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123", "--archive"})

	err := cmd.Execute()
	if err != nil {
		assert.NotContains(t, err.Error(), "no fields specified")
		assert.NotContains(t, err.Error(), "cannot be used together")
	}
}

func TestUpdateCmd_UnarchiveFlag_ValidationPasses(t *testing.T) {
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123", "--unarchive"})

	err := cmd.Execute()
	if err != nil {
		assert.NotContains(t, err.Error(), "no fields specified")
		assert.NotContains(t, err.Error(), "cannot be used together")
	}
}

func TestUpdateCmd_WithNameFlag_ValidationPasses(t *testing.T) {
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123", "--name", "New Task Name"})

	err := cmd.Execute()
	if err != nil {
		assert.NotContains(t, err.Error(), "no fields specified")
		assert.NotContains(t, err.Error(), "cannot be used together")
	}
}

func TestUpdateCmd_WithClearFlag_ValidationPasses(t *testing.T) {
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123", "--clear", "description"})

	err := cmd.Execute()
	if err != nil {
		assert.NotContains(t, err.Error(), "no fields specified")
		assert.NotContains(t, err.Error(), "cannot be used together")
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail (compile error)**

```
go test ./cmd/clickup/task/...
```

Expected: compile error — `undefined: lookupTaskType`, `undefined: newUpdateCmd`

- [ ] **Step 3: Create task/cmd.go**

```go
// cmd/clickup/task/cmd.go
package task

import "github.com/spf13/cobra"

func NewCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
	}
	cmd.AddCommand(newListCmd(configPath))
	cmd.AddCommand(newGetCmd(configPath))
	cmd.AddCommand(newCreateCmd(configPath))
	cmd.AddCommand(newUpdateCmd(configPath))
	cmd.AddCommand(newDeleteCmd(configPath))
	return cmd
}
```

- [ ] **Step 4: Create task/list.go**

```go
// cmd/clickup/task/list.go
package task

import (
	"context"
	"fmt"
	"time"

	"github.com/hiraking/click-up-cli/cmd/clickup/cmdutil"
	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/hiraking/click-up-cli/internal/dateparse"
	"github.com/spf13/cobra"
)

func newListCmd(configPath *string) *cobra.Command {
	var lists []string
	var statuses []string
	var dueAfterStr string
	var dueBeforeStr string
	var noSubtasks bool
	var query string
	var includeArchived bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get tasks as a JSON tree",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cmdutil.LoadConfig(*configPath)
			if err != nil {
				return err
			}

			var listIDs []string
			if len(lists) > 0 {
				listIDs = make([]string, 0, len(lists))
				for _, name := range lists {
					id, ok := cfg.Lists[name]
					if !ok {
						return fmt.Errorf("Error: Unknown list name '%s'. Available: %s",
							name, cmdutil.AvailableListNames(cfg.Lists))
					}
					listIDs = append(listIDs, id)
				}
			}

			var dueDateGt, dueDateLt *time.Time
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

			c := client.New(cfg.APIKey)
			tasks, err := c.GetTasks(context.Background(), cfg.TeamID, client.GetTasksOptions{
				IncludeSubtasks: !noSubtasks,
				ListIDs:         listIDs,
				Statuses:        statuses,
				DueDateGt:       dueDateGt,
				DueDateLt:       dueDateLt,
				Query:           query,
				IncludeArchived: includeArchived,
			})
			if err != nil {
				return err
			}
			return cmdutil.PrintJSON(tasks)
		},
	}

	cmd.Flags().StringArrayVar(&lists, "list", nil,
		"List name(s) defined in config.json (repeatable). Omit for all lists.")
	cmd.Flags().StringArrayVar(&statuses, "status", nil,
		"Status name(s) to filter by (repeatable), e.g. \"in progress\".")
	cmd.Flags().StringVar(&dueAfterStr, "due-after", "",
		"ISO 8601 datetime. Return only tasks with due date after this value.")
	cmd.Flags().StringVar(&dueBeforeStr, "due-before", "",
		"ISO 8601 datetime. Return only tasks with due date before this value.")
	cmd.Flags().BoolVar(&noSubtasks, "no-subtasks", false,
		"Exclude subtasks from results.")
	cmd.Flags().StringVar(&query, "query", "",
		"Case-insensitive substring to match against task name and description. Filtering is performed client-side after fetching all pages.")
	cmd.Flags().BoolVar(&includeArchived, "archived", false,
		"Return archived tasks only (non-archived tasks are excluded).")

	return cmd
}
```

- [ ] **Step 5: Create task/get.go**

```go
// cmd/clickup/task/get.go
package task

import (
	"context"

	"github.com/hiraking/click-up-cli/cmd/clickup/cmdutil"
	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/spf13/cobra"
)

func newGetCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <taskId>",
		Short: "Get a single task by ID as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cmdutil.LoadConfig(*configPath)
			if err != nil {
				return err
			}
			c := client.New(cfg.APIKey)
			task, err := c.GetTask(context.Background(), args[0])
			if err != nil {
				return err
			}
			return cmdutil.PrintJSON(task)
		},
	}
}
```

- [ ] **Step 6: Create task/create.go**

```go
// cmd/clickup/task/create.go
package task

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hiraking/click-up-cli/cmd/clickup/cmdutil"
	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/hiraking/click-up-cli/internal/dateparse"
	"github.com/hiraking/click-up-cli/internal/models"
	"github.com/spf13/cobra"
)

func newCreateCmd(configPath *string) *cobra.Command {
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
		Use:   "create <name>",
		Short: "Create a new task and output it as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := cmdutil.LoadConfig(*configPath)
			if err != nil {
				return err
			}

			listID, ok := cfg.Lists[listName]
			if !ok {
				return fmt.Errorf("Error: Unknown list name '%s'. Available: %s",
					listName, cmdutil.AvailableListNames(cfg.Lists))
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
			return cmdutil.PrintJSON(task)
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

- [ ] **Step 7: Create task/update.go**

```go
// cmd/clickup/task/update.go
package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hiraking/click-up-cli/cmd/clickup/cmdutil"
	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/hiraking/click-up-cli/internal/dateparse"
	"github.com/hiraking/click-up-cli/internal/models"
	"github.com/spf13/cobra"
)

var validClearFields = map[string]bool{
	"description":   true,
	"status":        true,
	"priority":      true,
	"due-date":      true,
	"start-date":    true,
	"time-estimate": true,
}

func newUpdateCmd(configPath *string) *cobra.Command {
	var name string
	var description string
	var status string
	var priority string
	var dueDateStr string
	var startDateStr string
	var timeEstimateMin int
	var parentID string
	var clearFields []string
	var archive bool
	var unarchive bool

	cmd := &cobra.Command{
		Use:   "update <taskId>",
		Short: "Update an existing task and output it as JSON",
		Long: `Update an existing ClickUp task by task ID.

Only the flags you specify will be updated. Flags not provided are left unchanged.

Clearing fields:
  Use --clear FIELD to remove a field's value entirely.
  Accepted values: description, status, priority, due-date, start-date, time-estimate

  Note: 'name' cannot be cleared (required field).
        'parent' cannot be cleared (ClickUp API does not support removing parent).

  Examples:
    task update abc123 --clear due-date
    task update abc123 --clear due-date --clear priority
    task update abc123 --name "New Name" --clear description`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			for _, f := range clearFields {
				if !validClearFields[f] {
					return fmt.Errorf("Error: invalid field for --clear: %q. Accepted: description, status, priority, due-date, start-date, time-estimate", f)
				}
			}

			changed := cmd.Flags().Changed
			if !changed("name") && !changed("description") && !changed("status") &&
				!changed("priority") && !changed("due-date") && !changed("start-date") &&
				!changed("time-estimate") && !changed("parent") && !changed("archive") &&
				!changed("unarchive") && len(clearFields) == 0 {
				return fmt.Errorf("Error: no fields specified to update.")
			}

			if changed("archive") && changed("unarchive") {
				return fmt.Errorf("Error: --archive and --unarchive cannot be used together.")
			}

			cfg, err := cmdutil.LoadConfig(*configPath)
			if err != nil {
				return err
			}

			req := models.UpdateTaskRequest{
				ClearFields: clearFields,
			}

			if changed("name") {
				req.Name = &name
			}
			if changed("description") {
				req.Description = &description
			}
			if changed("status") {
				req.Status = &status
			}
			if changed("priority") {
				p, err := parsePriority(priority)
				if err != nil {
					return err
				}
				req.Priority = &p
			}
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
			if changed("time-estimate") {
				d := time.Duration(timeEstimateMin) * time.Minute
				req.TimeEstimate = &d
			}
			if changed("parent") {
				if strings.TrimSpace(parentID) == "" {
					return fmt.Errorf("Error: '--parent' must not be empty or whitespace.")
				}
				req.Parent = &parentID
			}
			if changed("archive") {
				v := true
				req.Archived = &v
			}
			if changed("unarchive") {
				v := false
				req.Archived = &v
			}

			c := client.New(cfg.APIKey)
			task, err := c.UpdateTask(context.Background(), taskID, req)
			if err != nil {
				return err
			}
			return cmdutil.PrintJSON(task)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New task name.")
	cmd.Flags().StringVar(&description, "description", "", "New task description.")
	cmd.Flags().StringVar(&status, "status", "", "New status name (e.g. \"to do\", \"in progress\").")
	cmd.Flags().StringVar(&priority, "priority", "", "New priority: urgent, high, normal, or low.")
	cmd.Flags().StringVar(&dueDateStr, "due-date", "", "New due date as ISO 8601. Timezone-less values use the timezone from config (default UTC).")
	cmd.Flags().StringVar(&startDateStr, "start-date", "", "New start date as ISO 8601. Timezone-less values use the timezone from config (default UTC).")
	cmd.Flags().IntVar(&timeEstimateMin, "time-estimate", 0, "New time estimate in minutes.")
	cmd.Flags().StringVar(&parentID, "parent", "", "New parent task ID.")
	cmd.Flags().StringArrayVar(&clearFields, "clear", nil,
		"Field to clear (repeatable). Accepted: description, status, priority, due-date, start-date, time-estimate.\n"+
			"Use --clear FIELD to remove a field's value from the task.")
	cmd.Flags().BoolVar(&archive, "archive", false, "Archive the task.")
	cmd.Flags().BoolVar(&unarchive, "unarchive", false, "Unarchive the task.")

	return cmd
}
```

- [ ] **Step 8: Create task/delete.go**

```go
// cmd/clickup/task/delete.go
package task

import (
	"context"

	"github.com/hiraking/click-up-cli/cmd/clickup/cmdutil"
	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/spf13/cobra"
)

type deleteTaskResult struct {
	Deleted bool   `json:"deleted"`
	TaskID  string `json:"taskId"`
}

func newDeleteCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <taskId>",
		Short: "Delete a task by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			cfg, err := cmdutil.LoadConfig(*configPath)
			if err != nil {
				return err
			}
			c := client.New(cfg.APIKey)
			if err := c.DeleteTask(context.Background(), taskID); err != nil {
				return err
			}
			return cmdutil.PrintJSON(deleteTaskResult{Deleted: true, TaskID: taskID})
		},
	}
}
```

- [ ] **Step 9: Run tests to confirm they pass**

```
go test ./cmd/clickup/task/...
```

Expected: `ok  github.com/hiraking/click-up-cli/cmd/clickup/task`

- [ ] **Step 10: Commit**

```
git add cmd/clickup/task/
git commit -m "feat: add task subcommand package (list/get/create/update/delete)"
```

---

## Task 3: Create time package

**Files:**
- Create: `cmd/clickup/time/cmd.go`
- Create: `cmd/clickup/time/report.go`

- [ ] **Step 1: Create time/cmd.go**

```go
// cmd/clickup/time/cmd.go
package timecmd

import "github.com/spf13/cobra"

func NewCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "time",
		Short: "Manage time entries",
	}
	cmd.AddCommand(newReportCmd(configPath))
	return cmd
}
```

- [ ] **Step 2: Create time/report.go**

```go
// cmd/clickup/time/report.go
package timecmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/hiraking/click-up-cli/cmd/clickup/cmdutil"
	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/hiraking/click-up-cli/internal/dateparse"
	"github.com/hiraking/click-up-cli/internal/timereport"
	"github.com/spf13/cobra"
)

func newReportCmd(configPath *string) *cobra.Command {
	var flagStart, flagEnd, flagOutput string
	var flagRows bool

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Aggregate time entries and output a JSON report",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cmdutil.LoadConfig(*configPath)
			if err != nil {
				return err
			}

			start, err := dateparse.ParseISO(flagStart, "start", cfg.TimezoneLocation())
			if err != nil {
				return err
			}
			end, err := dateparse.ParseISO(flagEnd, "end", cfg.TimezoneLocation())
			if err != nil {
				return err
			}
			if !end.After(start) {
				return fmt.Errorf("--end must be after --start")
			}

			ctx := context.Background()
			c := client.New(cfg.APIKey)

			entries, err := c.GetTimeEntries(ctx, cfg.TeamID, client.GetTimeEntriesOptions{
				Start: start,
				End:   end,
			})
			if err != nil {
				return err
			}

			report, err := timereport.Build(ctx, entries, start, end, c.GetTask)
			if err != nil {
				return err
			}

			includeRows := flagOutput != ""
			if cmd.Flags().Changed("rows") {
				includeRows = flagRows
			}
			if !includeRows {
				report.Rows = nil
			}

			var w io.Writer = os.Stdout
			if flagOutput != "" {
				f, err := os.Create(flagOutput)
				if err != nil {
					return fmt.Errorf("failed to create output file: %w", err)
				}
				defer f.Close()
				w = f
			}

			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			enc.SetEscapeHTML(false)
			return enc.Encode(report)
		},
	}

	cmd.Flags().StringVar(&flagStart, "start", "", "Report start datetime (ISO 8601, inclusive)")
	cmd.Flags().StringVar(&flagEnd, "end", "", "Report end datetime (ISO 8601, exclusive)")
	cmd.Flags().StringVarP(&flagOutput, "output", "o", "", "Output file path (default: stdout)")
	cmd.Flags().BoolVar(&flagRows, "rows", false, "Include normalized rows in output")

	_ = cmd.MarkFlagRequired("start")
	_ = cmd.MarkFlagRequired("end")

	return cmd
}
```

- [ ] **Step 3: Verify the package compiles**

```
go build ./cmd/clickup/time/...
```

Expected: no output (success)

- [ ] **Step 4: Commit**

```
git add cmd/clickup/time/
git commit -m "feat: add time subcommand package (report)"
```

---

## Task 4: Create config package

**Files:**
- Create: `cmd/clickup/config/cmd.go`
- Create: `cmd/clickup/config/show.go`

- [ ] **Step 1: Create config/cmd.go**

```go
// cmd/clickup/config/cmd.go
package configcmd

import "github.com/spf13/cobra"

func NewCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}
	cmd.AddCommand(newShowCmd(configPath))
	return cmd
}
```

- [ ] **Step 2: Create config/show.go**

```go
// cmd/clickup/config/show.go
package configcmd

import (
	"github.com/hiraking/click-up-cli/cmd/clickup/cmdutil"
	"github.com/spf13/cobra"
)

func newShowCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cmdutil.LoadConfig(*configPath)
			if err != nil {
				return err
			}
			lists := cfg.Lists
			if lists == nil {
				lists = map[string]string{}
			}
			out := struct {
				APIKey   string            `json:"apiKey"`
				TeamID   string            `json:"teamId"`
				Lists    map[string]string `json:"lists"`
				Timezone string            `json:"timezone,omitempty"`
			}{
				APIKey:   cmdutil.MaskAPIKey(cfg.APIKey),
				TeamID:   cfg.TeamID,
				Lists:    lists,
				Timezone: cfg.Timezone,
			}
			return cmdutil.PrintJSON(out)
		},
	}
}
```

- [ ] **Step 3: Verify the package compiles**

```
go build ./cmd/clickup/config/...
```

Expected: no output (success)

- [ ] **Step 4: Commit**

```
git add cmd/clickup/config/
git commit -m "feat: add config subcommand package (show)"
```

---

## Task 5: Rewrite main.go

**Files:**
- Modify: `cmd/clickup/main.go`

- [ ] **Step 1: Replace the contents of main.go**

```go
// cmd/clickup/main.go
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	configcmd "github.com/hiraking/click-up-cli/cmd/clickup/config"
	"github.com/hiraking/click-up-cli/cmd/clickup/task"
	timecmd "github.com/hiraking/click-up-cli/cmd/clickup/time"
	"github.com/spf13/cobra"
)

var version = "dev"

func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

func main() {
	var configPath string

	rootCmd := &cobra.Command{
		Use:           "clickup",
		Short:         "ClickUp API CLI wrapper",
		Version:       resolveVersion(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path (default: ~/.clickup/config.json)")

	rootCmd.AddCommand(task.NewCmd(&configPath))
	rootCmd.AddCommand(timecmd.NewCmd(&configPath))
	rootCmd.AddCommand(configcmd.NewCmd(&configPath))

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify the binary builds (old files still exist alongside new packages — no conflict)**

```
go build ./cmd/clickup/...
```

Expected: no output (success)

- [ ] **Step 3: Commit**

```
git add cmd/clickup/main.go
git commit -m "refactor: wire new resource subcommands in main.go"
```

---

## Task 6: Remove old source files and verify

**Files:**
- Delete: `cmd/clickup/get_tasks.go`
- Delete: `cmd/clickup/get_task.go`
- Delete: `cmd/clickup/create_task.go`
- Delete: `cmd/clickup/create_task_test.go`
- Delete: `cmd/clickup/update_task.go`
- Delete: `cmd/clickup/update_task_test.go`
- Delete: `cmd/clickup/delete_task.go`
- Delete: `cmd/clickup/time_report.go`
- Delete: `cmd/clickup/show_config.go`
- Delete: `cmd/clickup/helpers.go`
- Delete: `cmd/clickup/helpers_test.go`

- [ ] **Step 1: Delete old files**

```
git rm cmd/clickup/get_tasks.go cmd/clickup/get_task.go cmd/clickup/create_task.go cmd/clickup/create_task_test.go cmd/clickup/update_task.go cmd/clickup/update_task_test.go cmd/clickup/delete_task.go cmd/clickup/time_report.go cmd/clickup/show_config.go cmd/clickup/helpers.go cmd/clickup/helpers_test.go
```

- [ ] **Step 2: Run the full test suite**

```
go test ./...
```

Expected:
```
ok  github.com/hiraking/click-up-cli/cmd/clickup/cmdutil
ok  github.com/hiraking/click-up-cli/cmd/clickup/task
ok  github.com/hiraking/click-up-cli/internal/...
```
(All packages pass, no compile errors.)

- [ ] **Step 3: Build the binary**

```
go build -o clickup.exe ./cmd/clickup/
```

Expected: no output (success). The binary is produced.

- [ ] **Step 4: Smoke-test the binary**

```
.\clickup.exe --help
.\clickup.exe task --help
.\clickup.exe time --help
.\clickup.exe config --help
```

Expected: Each help output lists the correct subcommands (`list`, `get`, `create`, `update`, `delete` for `task`; `report` for `time`; `show` for `config`).

- [ ] **Step 5: Commit**

```
git add -A
git commit -m "refactor: remove old flat command files"
```

---

## Task 7: Update README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Replace the `## Commands` section in README.md**

Replace the entire `## Commands` section (from `### \`get-tasks\`` through the end of `### \`show-config\``) with the following:

````markdown
## Commands

### `task list`

Fetches tasks as a tree.

```
clickup task list [options]
```

| Option | Type | Description |
|---|---|---|
| `--list <name>` | string | List name from `config.json`. Repeatable. Defaults to all lists. |
| `--status <name>` | string | Filter by status. Repeatable. |
| `--due-after <ISO8601>` | string | Only tasks with a due date after this datetime. Timezone-less values use the `timezone` from config (default UTC). |
| `--due-before <ISO8601>` | string | Only tasks with a due date before this datetime. Timezone-less values use the `timezone` from config (default UTC). |
| `--no-subtasks` | flag | Exclude subtasks (default: included) |
| `--archived` | flag | Return archived tasks only (non-archived tasks are excluded). |
| `--query <text>` | string | Case-insensitive substring filter on task name and description (client-side, applied after all pages are fetched) |

**Output:** JSON array of root tasks. Subtasks are nested under each task's `subtasks` field.

```json
[
  {
    "id": "86exa7yq5",
    "name": "Write design doc",
    "status": "in progress",
    "priority": "high",
    "parentId": null,
    "subtasks": [
      {
        "id": "86exa8ab3",
        "name": "Draft outline",
        "status": "to do",
        "parentId": "86exa7yq5",
        "subtasks": []
      }
    ]
  }
]
```

> - Automatically paginates up to 10 pages (1,000 tasks). If the limit is reached, a warning is printed to stderr and the fetched results are returned.
> - `--due-after` / `--due-before` filtering is handled server-side by the ClickUp API.
> - **Timezone:** Datetime strings without an offset (e.g. `"2026-05-01"`, `"2026-05-01T09:00"`) are interpreted using the `timezone` setting from config (default: UTC). Explicit offsets (e.g. `"2026-05-01T00:00:00Z"`) are used as-is.

```bash
clickup task list
clickup task list --list work
clickup task list --list work --list study
clickup task list --list work --status active
clickup task list --due-before 2026-04-21T23:59:59+09:00
clickup task list --list work --no-subtasks
clickup task list --archived
clickup task list --query "design"
```

---

### `task create`

Creates a new task.

```
clickup task create <name> --list <name> [options]
```

| Argument/Option | Type | Description |
|---|---|---|
| `name` | string | Task name (required) |
| `--list <name>` | string | Destination list name (required) |
| `--description <text>` | string | Task description |
| `--parent <taskId>` | string | Parent task ID (creates a subtask) |
| `--status <name>` | string | Status name (e.g. `"to do"`, `"in progress"`) |
| `--priority <value>` | string | `urgent` / `high` / `normal` / `low` |
| `--due-date <ISO8601>` | string | Due date |
| `--start-date <ISO8601>` | string | Start date |
| `--time-estimate <minutes>` | int | Time estimate in minutes |
| `--task-type <name>` | string | Task type name as defined in `taskTypes` config (case-sensitive). |

**Output:** `TaskSummary` object of the created task (same shape as `task get`).

```bash
clickup task create "My task" --list work

clickup task create "Write design doc" --list work \
  --description "Architecture design" \
  --parent "86exa7yq5" \
  --status "to do" \
  --priority high \
  --due-date "2026-05-01T18:00+09:00" \
  --start-date "2026-04-25T09:00" \
  --time-estimate 120

clickup task create "Q2 Plan" --list work --task-type milestone
```

---

### `task get`

Fetches a single task by ID.

```
clickup task get <taskId>
```

```bash
clickup task get 86exa7yq5
```

```json
{
  "id": "86exa7yq5",
  "name": "English study",
  "status": "active",
  "priority": null,
  "parentId": null,
  "url": "https://app.clickup.com/t/86exa7yq5",
  "dueDate": null,
  "description": "",
  "listId": "900523741862",
  "listName": "Study",
  "createdAt": "2026-04-19T15:09:41.393Z",
  "updatedAt": "2026-04-19T16:05:33.346Z",
  "subtasks": []
}
```

---

### `task update`

Updates a task. Only specified fields are changed.

```
clickup task update <taskId> [options]
```

| Option | Type | Description |
|---|---|---|
| `--name <text>` | string | New task name |
| `--description <text>` | string | New description |
| `--status <name>` | string | New status |
| `--priority <value>` | string | `urgent` / `high` / `normal` / `low` |
| `--due-date <ISO8601>` | string | New due date |
| `--start-date <ISO8601>` | string | New start date |
| `--time-estimate <minutes>` | int | New time estimate in minutes |
| `--parent <taskId>` | string | New parent task ID |
| `--clear <field>` | string | Clear a field (repeatable). Clearable fields: `description`, `status`, `priority`, `due-date`, `start-date`, `time-estimate` |
| `--archive` | flag | Archive the task |
| `--unarchive` | flag | Unarchive the task |

> `name` and `parent` cannot be cleared (`name` is required by the API; removing a subtask's parent is not supported).
> `--archive` and `--unarchive` are mutually exclusive and cannot be used together.

**Output:** `TaskSummary` object of the updated task (same shape as `task get`).

```bash
clickup task update 86exa7yq5 --name "New name"
clickup task update 86exa7yq5 --status "in progress" --priority high
clickup task update 86exa7yq5 --clear due-date
clickup task update 86exa7yq5 --name "New name" --clear description
clickup task update 86exa7yq5 --clear due-date --clear priority
clickup task update 86exa7yq5 --archive
clickup task update 86exa7yq5 --unarchive
clickup task update 86exa7yq5 --archive --status done
```

---

### `task delete`

Deletes a task by ID. No confirmation is required.

```
clickup task delete <taskId>
```

**Output:**

```json
{
  "deleted": true,
  "taskId": "86exa7yq5"
}
```

```bash
clickup task delete 86exa7yq5
```

---

### `time report`

Aggregates time entries for a period and outputs a JSON report grouped by List → Task → Breakdown.

```
clickup time report --start <ISO8601> --end <ISO8601> [options]
```

| Option | Type | Description |
|---|---|---|
| `--start <ISO8601>` | string | Start of the period (required, inclusive) |
| `--end <ISO8601>` | string | End of the period (required, exclusive) |
| `--output`, `-o <path>` | string | Output file path. Defaults to stdout. |
| `--rows` | bool | Include normalized rows. Defaults to `true` when `--output` is set, `false` otherwise. |

```bash
# Weekly report to stdout (no rows)
clickup time report \
  --start "2026-04-27T00:00:00+09:00" \
  --end   "2026-05-04T00:00:00+09:00"

# Save to file (rows included by default)
clickup time report \
  --start "2026-04-27T00:00:00+09:00" \
  --end   "2026-05-04T00:00:00+09:00" \
  --output report.json

# Save to file, explicitly exclude rows
clickup time report \
  --start "2026-04-27T00:00:00+09:00" \
  --end   "2026-05-04T00:00:00+09:00" \
  --output report.json \
  --rows=false
```

**Output** (without `--rows`):

```json
{
  "generatedAt": "2026-05-07T10:00:00Z",
  "period": {
    "start": "2026-04-27T00:00:00+09:00",
    "end": "2026-05-04T00:00:00+09:00",
    "timezone": "Asia/Tokyo"
  },
  "summary": {
    "totalDurationMin": 370,
    "listCount": 2,
    "topLevelTaskCount": 3,
    "breakdownTaskCount": 2
  },
  "hierarchy": [
    {
      "listId": "900523741862",
      "listName": "Work",
      "durationMin": 270,
      "tasks": [
        {
          "taskId": "86exa7yq5",
          "taskName": "Write design doc",
          "durationMin": 150,
          "breakdown": [
            { "taskId": "86exa7yq5", "taskName": "Write design doc", "durationMin": 90 },
            { "taskId": "86exa8ab3", "taskName": "Draft outline", "durationMin": 60 }
          ]
        },
        {
          "taskId": "86exb1cd2",
          "taskName": "Code review",
          "durationMin": 120
        }
      ]
    },
    {
      "listId": "900523741899",
      "listName": "Study",
      "durationMin": 100,
      "tasks": [
        {
          "taskId": "86exc3ef4",
          "taskName": "English study",
          "durationMin": 100
        }
      ]
    }
  ]
}
```

> - `breakdown` is omitted when all time was recorded directly on the top-level task (no subtask entries).
> - `rows` field is included only when `--rows` is enabled. Each row represents one clipped time entry with `originalStart/End` and `clippedStart/End`.

---

### `config show`

Prints the resolved configuration as JSON. The API key is masked, showing only the last 4 characters.

```
clickup config show
clickup --config /path/to/other-config.json config show
```

```json
{
  "apiKey": "****a1b2",
  "teamId": "90371842016",
  "lists": {
    "work": "900523741862",
    "study": "900523741899"
  },
  "timezone": "Asia/Tokyo"
}
```

> `timezone` is omitted from the output if not set in config.
````

- [ ] **Step 2: Commit**

```
git add README.md
git commit -m "docs: update README for new CLI hierarchy (task/time/config)"
```
