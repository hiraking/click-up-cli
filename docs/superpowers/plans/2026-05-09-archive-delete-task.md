# Archive and Delete Task Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--archive`/`--unarchive` flags to `update-task` and a new `delete-task` command.

**Architecture:** Archive is an extension of the existing `UpdateTaskRequest` model and `mapToRawUpdateBody` mapper — `Archived *bool` is added and sent as `{"archived": true/false}` to `PUT /v2/task/{id}`. Delete is a new `DeleteTask` method on `ClickUpClient` that calls `DELETE /v2/task/{id}` and a new `delete-task` cobra command.

**Tech Stack:** Go, cobra (CLI), standard library only for `internal/`

---

## File Map

| File | Action | Purpose |
|---|---|---|
| `internal/models/create_task.go` | Modify | Add `Archived *bool` to `UpdateTaskRequest` |
| `internal/client/mapper.go` | Modify | Handle `Archived` in `mapToRawUpdateBody` |
| `internal/client/mapper_test.go` | Modify | Tests for `Archived` in `mapToRawUpdateBody` |
| `internal/client/client.go` | Modify | Add `DeleteTask` to `ClickUpClient` interface; implement on `httpClient` |
| `cmd/clickup/update_task.go` | Modify | Add `--archive`/`--unarchive` flags, mutual exclusion, pass to req |
| `cmd/clickup/delete_task.go` | Create | New `delete-task` command |
| `cmd/clickup/main.go` | Modify | Register `newDeleteTaskCmd()` |
| `README.md` | Modify | Document new flags and command |

---

## Task 1: Add `Archived` to `UpdateTaskRequest` and mapper

**Files:**
- Modify: `internal/models/create_task.go`
- Modify: `internal/client/mapper.go`
- Modify: `internal/client/mapper_test.go`

- [ ] **Step 1: Write failing tests for `mapToRawUpdateBody` with `Archived`**

Append to `internal/client/mapper_test.go`:

```go
func TestMapToRawUpdateBody_Archive(t *testing.T) {
	archived := true
	req := models.UpdateTaskRequest{Archived: &archived}

	body := mapToRawUpdateBody(req)

	assert.Equal(t, true, body["archived"])
}

func TestMapToRawUpdateBody_Unarchive(t *testing.T) {
	archived := false
	req := models.UpdateTaskRequest{Archived: &archived}

	body := mapToRawUpdateBody(req)

	assert.Equal(t, false, body["archived"])
}

func TestMapToRawUpdateBody_ArchivedNil_NotInBody(t *testing.T) {
	req := models.UpdateTaskRequest{}

	body := mapToRawUpdateBody(req)

	_, ok := body["archived"]
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```
go test ./internal/client/... -run TestMapToRawUpdateBody_Archive -v
```

Expected: compile error — `models.UpdateTaskRequest` has no field `Archived`

- [ ] **Step 3: Add `Archived *bool` to `UpdateTaskRequest`**

In `internal/models/create_task.go`, add `Archived *bool` to `UpdateTaskRequest`:

```go
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
	Archived     *bool
	ClearFields  []string
}
```

- [ ] **Step 4: Handle `Archived` in `mapToRawUpdateBody`**

In `internal/client/mapper.go`, add after the `req.Parent` block (before the `ClearFields` loop):

```go
	if req.Archived != nil {
		body["archived"] = *req.Archived
	}
```

Full updated function for reference:
```go
func mapToRawUpdateBody(req models.UpdateTaskRequest) map[string]interface{} {
	body := make(map[string]interface{})

	if req.Name != nil {
		body["name"] = *req.Name
	}
	if req.Description != nil {
		body["description"] = *req.Description
	}
	if req.Status != nil {
		body["status"] = *req.Status
	}
	if req.Priority != nil {
		body["priority"] = int(*req.Priority)
	}
	if req.DueDate != nil {
		body["due_date"] = req.DueDate.UnixMilli()
		body["due_date_time"] = hasTimeComponent(*req.DueDate)
	}
	if req.StartDate != nil {
		body["start_date"] = req.StartDate.UnixMilli()
		body["start_date_time"] = hasTimeComponent(*req.StartDate)
	}
	if req.TimeEstimate != nil {
		body["time_estimate"] = int(req.TimeEstimate.Milliseconds())
	}
	if req.Parent != nil {
		body["parent"] = *req.Parent
	}
	if req.Archived != nil {
		body["archived"] = *req.Archived
	}

	for _, field := range req.ClearFields {
		switch field {
		case "description":
			body["description"] = " "
		case "status":
			body["status"] = nil
		case "priority":
			body["priority"] = nil
		case "due-date":
			body["due_date"] = nil
			delete(body, "due_date_time")
		case "start-date":
			body["start_date"] = nil
			delete(body, "start_date_time")
		case "time-estimate":
			body["time_estimate"] = nil
		}
	}

	return body
}
```

- [ ] **Step 5: Run tests to verify they pass**

```
go test ./internal/client/... -run TestMapToRawUpdateBody -v
```

Expected: all `TestMapToRawUpdateBody_*` tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/models/create_task.go internal/client/mapper.go internal/client/mapper_test.go
git commit -m "feat: add Archived field to UpdateTaskRequest and mapper"
```

---

## Task 2: Add `--archive`/`--unarchive` flags to `update-task`

**Files:**
- Modify: `cmd/clickup/update_task.go`

- [ ] **Step 1: Add `archive` and `unarchive` bool variables and flags**

In `cmd/clickup/update_task.go`, add the two variables alongside the existing ones:

```go
var archive bool
var unarchive bool
```

Add the flags at the end of the `cmd.Flags()` block (before `return cmd`):

```go
cmd.Flags().BoolVar(&archive, "archive", false, "Archive the task.")
cmd.Flags().BoolVar(&unarchive, "unarchive", false, "Unarchive the task.")
```

- [ ] **Step 2: Add mutual exclusion validation and include flags in the "no fields" check**

In the `RunE` function, update the "no fields specified" check to include archive/unarchive:

```go
if !changed("name") && !changed("description") && !changed("status") &&
    !changed("priority") && !changed("due-date") && !changed("start-date") &&
    !changed("time-estimate") && !changed("parent") && !changed("archive") &&
    !changed("unarchive") && len(clearFields) == 0 {
    return fmt.Errorf("Error: no fields specified to update.")
}
```

Add mutual exclusion check immediately after (before `cfg, err := loadConfig()`):

```go
if changed("archive") && changed("unarchive") {
    return fmt.Errorf("Error: --archive and --unarchive cannot be used together.")
}
```

- [ ] **Step 3: Pass `Archived` to the request**

Add after the existing `if changed("parent")` block:

```go
if changed("archive") {
    v := true
    req.Archived = &v
}
if changed("unarchive") {
    v := false
    req.Archived = &v
}
```

- [ ] **Step 4: Run all tests to confirm nothing is broken**

```
go test ./...
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/clickup/update_task.go
git commit -m "feat: add --archive/--unarchive flags to update-task"
```

---

## Task 3: Add `DeleteTask` to `ClickUpClient` and implement it

**Files:**
- Modify: `internal/client/client.go`

- [ ] **Step 1: Add `DeleteTask` to the `ClickUpClient` interface**

In `internal/client/client.go`, update the `ClickUpClient` interface:

```go
type ClickUpClient interface {
	GetTasks(ctx context.Context, teamID string, opts GetTasksOptions) ([]models.TaskSummary, error)
	GetTask(ctx context.Context, taskID string) (models.TaskSummary, error)
	CreateTask(ctx context.Context, listID string, req models.CreateTaskRequest) (models.TaskSummary, error)
	UpdateTask(ctx context.Context, taskID string, req models.UpdateTaskRequest) (models.TaskSummary, error)
	DeleteTask(ctx context.Context, taskID string) error
	GetTimeEntries(ctx context.Context, teamID string, opts GetTimeEntriesOptions) ([]models.TimeEntry, error)
}
```

- [ ] **Step 2: Implement `DeleteTask` on `httpClient`**

Add after the `UpdateTask` method:

```go
func (c *httpClient) DeleteTask(ctx context.Context, taskID string) error {
	rawURL := c.base + "v2/task/" + taskID

	respBody, status, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", c.apiKey)
		return req, nil
	})
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return ErrNotFound
	}
	if status >= 400 {
		return fmt.Errorf("HTTP Error (%d): %s", status, string(respBody))
	}
	return nil
}
```

- [ ] **Step 3: Run all tests to confirm the build succeeds**

```
go test ./...
```

Expected: all tests PASS (no compile errors — `httpClient` now satisfies the updated interface)

- [ ] **Step 4: Commit**

```bash
git add internal/client/client.go
git commit -m "feat: add DeleteTask method to ClickUpClient"
```

---

## Task 4: Add `delete-task` command

**Files:**
- Create: `cmd/clickup/delete_task.go`
- Modify: `cmd/clickup/main.go`

- [ ] **Step 1: Create `cmd/clickup/delete_task.go`**

```go
// cmd/clickup/delete_task.go
package main

import (
	"context"

	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/spf13/cobra"
)

type deleteTaskResult struct {
	Deleted bool   `json:"deleted"`
	TaskID  string `json:"taskId"`
}

func newDeleteTaskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete-task <taskId>",
		Short: "Delete a task by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			c := client.New(cfg.APIKey)
			if err := c.DeleteTask(context.Background(), taskID); err != nil {
				return err
			}
			return printJSON(deleteTaskResult{Deleted: true, TaskID: taskID})
		},
	}
}
```

- [ ] **Step 2: Register `newDeleteTaskCmd()` in `main.go`**

In `cmd/clickup/main.go`, add after `rootCmd.AddCommand(newUpdateTaskCmd())`:

```go
rootCmd.AddCommand(newDeleteTaskCmd())
```

- [ ] **Step 3: Run all tests to confirm the build succeeds**

```
go test ./...
```

Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/clickup/delete_task.go cmd/clickup/main.go
git commit -m "feat: add delete-task command"
```

---

## Task 5: Update README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update the `update-task` options table**

In the `update-task` section, add two rows to the options table:

```markdown
| `--archive` | flag | Archive the task |
| `--unarchive` | flag | Unarchive the task |
```

Also add examples:

```bash
clickup update-task 86exa7yq5 --archive
clickup update-task 86exa7yq5 --unarchive
clickup update-task 86exa7yq5 --archive --status done
```

- [ ] **Step 2: Add `delete-task` section**

Add a new section after `update-task`:

```markdown
### `delete-task`

Deletes a task by ID. No confirmation is required.

```
clickup delete-task <taskId>
```

**Output:**

```json
{
  "deleted": true,
  "taskId": "86exa7yq5"
}
```

```bash
clickup delete-task 86exa7yq5
```
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add --archive/--unarchive and delete-task to README"
```
