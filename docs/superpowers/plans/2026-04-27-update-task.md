# update-task コマンド 実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ClickUp タスクをフィールド単位で更新・クリアできる `update-task <taskId>` CLIコマンドを追加する。

**Architecture:** 既存の `create-task` コマンドと同じ `Changed()` パターンでフラグを処理する。フィールドのクリアには `--clear FIELD` フラグを使い、更新ボディは `map[string]interface{}` で組み立てて明示的 null を表現する。`description` のクリアのみ ClickUp API の仕様により `" "` (スペース) を送信する。

**Tech Stack:** Go 1.21+, github.com/spf13/cobra, github.com/stretchr/testify, ClickUp API v2

---

## ファイル構成

| ファイル | 操作 | 役割 |
|---|---|---|
| `internal/models/create_task.go` | 変更 | `UpdateTaskRequest` 型を追加 |
| `internal/client/mapper.go` | 変更 | `mapToRawUpdateBody()` 関数を追加 |
| `internal/client/mapper_test.go` | 変更 | `mapToRawUpdateBody` のテストを追加 |
| `internal/client/client.go` | 変更 | `UpdateTask()` をインターフェースと実装に追加 |
| `cmd/clickup/update_task.go` | 新規 | `newUpdateTaskCmd()` コマンド定義 |
| `cmd/clickup/main.go` | 変更 | `update-task` コマンドを登録 |
| `README.md` | 変更 | `update-task` のドキュメントを追加 |

---

## Task 1: UpdateTaskRequest モデルの追加

**Files:**
- Modify: `internal/models/create_task.go`

- [ ] **Step 1: `UpdateTaskRequest` 型を追加する**

`internal/models/create_task.go` の末尾に以下を追加:

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
	ClearFields  []string
}
```

- [ ] **Step 2: コンパイルを確認する**

```
go build ./...
```

期待: エラーなし

- [ ] **Step 3: コミットする**

```bash
git add internal/models/create_task.go
git commit -m "feat: add UpdateTaskRequest model"
```

---

## Task 2: mapToRawUpdateBody の実装とテスト

**Files:**
- Modify: `internal/client/mapper.go`
- Modify: `internal/client/mapper_test.go`

- [ ] **Step 1: テストを書く**

`internal/client/mapper_test.go` の末尾に以下を追加:

```go
func TestMapToRawUpdateBody_SetName(t *testing.T) {
	name := "New Name"
	req := models.UpdateTaskRequest{Name: &name}

	body := mapToRawUpdateBody(req)

	assert.Equal(t, "New Name", body["name"])
	assert.NotContains(t, body, "description")
}

func TestMapToRawUpdateBody_SetPriority(t *testing.T) {
	pri := models.PriorityHigh
	req := models.UpdateTaskRequest{Priority: &pri}

	body := mapToRawUpdateBody(req)

	assert.Equal(t, 2, body["priority"])
}

func TestMapToRawUpdateBody_SetDueDate_WithTime(t *testing.T) {
	due := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	req := models.UpdateTaskRequest{DueDate: &due}

	body := mapToRawUpdateBody(req)

	assert.Equal(t, due.UnixMilli(), body["due_date"])
	assert.Equal(t, true, body["due_date_time"])
}

func TestMapToRawUpdateBody_SetDueDate_Midnight(t *testing.T) {
	due := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	req := models.UpdateTaskRequest{DueDate: &due}

	body := mapToRawUpdateBody(req)

	assert.Equal(t, due.UnixMilli(), body["due_date"])
	assert.Equal(t, false, body["due_date_time"])
}

func TestMapToRawUpdateBody_SetTimeEstimate(t *testing.T) {
	d := 30 * time.Minute
	req := models.UpdateTaskRequest{TimeEstimate: &d}

	body := mapToRawUpdateBody(req)

	assert.Equal(t, int(d.Milliseconds()), body["time_estimate"])
}

func TestMapToRawUpdateBody_ClearDescription(t *testing.T) {
	// ClickUp API: description のクリアはスペース " " を送信する
	req := models.UpdateTaskRequest{ClearFields: []string{"description"}}

	body := mapToRawUpdateBody(req)

	assert.Equal(t, " ", body["description"])
}

func TestMapToRawUpdateBody_ClearPriority(t *testing.T) {
	req := models.UpdateTaskRequest{ClearFields: []string{"priority"}}

	body := mapToRawUpdateBody(req)

	assert.Nil(t, body["priority"])
	_, exists := body["priority"]
	assert.True(t, exists, "priority キーは存在するが値が nil であること")
}

func TestMapToRawUpdateBody_ClearDueDate(t *testing.T) {
	req := models.UpdateTaskRequest{ClearFields: []string{"due-date"}}

	body := mapToRawUpdateBody(req)

	assert.Nil(t, body["due_date"])
	_, exists := body["due_date"]
	assert.True(t, exists)
	assert.NotContains(t, body, "due_date_time")
}

func TestMapToRawUpdateBody_ClearStartDate(t *testing.T) {
	req := models.UpdateTaskRequest{ClearFields: []string{"start-date"}}

	body := mapToRawUpdateBody(req)

	assert.Nil(t, body["start_date"])
	_, exists := body["start_date"]
	assert.True(t, exists)
	assert.NotContains(t, body, "start_date_time")
}

func TestMapToRawUpdateBody_ClearTimeEstimate(t *testing.T) {
	req := models.UpdateTaskRequest{ClearFields: []string{"time-estimate"}}

	body := mapToRawUpdateBody(req)

	assert.Nil(t, body["time_estimate"])
	_, exists := body["time_estimate"]
	assert.True(t, exists)
}

func TestMapToRawUpdateBody_ClearStatus(t *testing.T) {
	req := models.UpdateTaskRequest{ClearFields: []string{"status"}}

	body := mapToRawUpdateBody(req)

	assert.Nil(t, body["status"])
	_, exists := body["status"]
	assert.True(t, exists)
}

func TestMapToRawUpdateBody_SetAndClear_ClearWins(t *testing.T) {
	// set と clear が同時に指定された場合、clear が優先される
	desc := "some text"
	req := models.UpdateTaskRequest{
		Description: &desc,
		ClearFields: []string{"description"},
	}

	body := mapToRawUpdateBody(req)

	assert.Equal(t, " ", body["description"])
}

func TestMapToRawUpdateBody_NoFields(t *testing.T) {
	req := models.UpdateTaskRequest{}

	body := mapToRawUpdateBody(req)

	assert.Empty(t, body)
}
```

- [ ] **Step 2: テストが失敗することを確認する**

```
go test ./internal/client/... -run TestMapToRawUpdateBody -v
```

期待: `undefined: mapToRawUpdateBody` でコンパイルエラー

- [ ] **Step 3: `mapToRawUpdateBody` を実装する**

`internal/client/mapper.go` に以下を追加（`mapToRawCreateBody` の後ろ）:

```go
// mapToRawUpdateBody は models.UpdateTaskRequest を PUT /v2/task/{taskId} ボディに変換する。
// map[string]interface{} を使うことでクリアフィールドへの明示的 null 送信を実現する。
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

- [ ] **Step 4: テストを実行してパスすることを確認する**

```
go test ./internal/client/... -run TestMapToRawUpdateBody -v
```

期待: 全テスト PASS

- [ ] **Step 5: 全テストが壊れていないことを確認する**

```
go test ./...
```

期待: 全テスト PASS

- [ ] **Step 6: コミットする**

```bash
git add internal/client/mapper.go internal/client/mapper_test.go
git commit -m "feat: add mapToRawUpdateBody with clear field support"
```

---

## Task 3: ClickUpClient インターフェースと UpdateTask メソッドの追加

**Files:**
- Modify: `internal/client/client.go`

- [ ] **Step 1: インターフェースに `UpdateTask` を追加する**

`internal/client/client.go` の `ClickUpClient` インターフェースを以下に変更:

```go
type ClickUpClient interface {
	GetTasks(ctx context.Context, teamID string, opts GetTasksOptions) ([]models.TaskSummary, error)
	GetTask(ctx context.Context, taskID string) (models.TaskSummary, error)
	CreateTask(ctx context.Context, listID string, req models.CreateTaskRequest) (models.TaskSummary, error)
	UpdateTask(ctx context.Context, taskID string, req models.UpdateTaskRequest) (models.TaskSummary, error)
}
```

- [ ] **Step 2: `UpdateTask` メソッドを実装する**

`internal/client/client.go` の `CreateTask` メソッドの後ろに以下を追加:

```go
func (c *httpClient) UpdateTask(ctx context.Context, taskID string, req models.UpdateTaskRequest) (models.TaskSummary, error) {
	rawURL := c.base + "v2/task/" + taskID
	body := mapToRawUpdateBody(req)

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, rawURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return models.TaskSummary{}, err
	}
	httpReq.Header.Set("Authorization", c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return models.TaskSummary{}, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		b, _ := io.ReadAll(httpResp.Body)
		return models.TaskSummary{}, fmt.Errorf("HTTP Error (%d): %s", httpResp.StatusCode, string(b))
	}

	var raw rawTask
	if err := json.NewDecoder(httpResp.Body).Decode(&raw); err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to decode response: %w", err)
	}
	return toSummary(raw), nil
}
```

- [ ] **Step 3: コンパイルと全テストを確認する**

```
go build ./...
go test ./...
```

期待: エラーなし、全テスト PASS

- [ ] **Step 4: コミットする**

```bash
git add internal/client/client.go
git commit -m "feat: add UpdateTask method to ClickUpClient"
```

---

## Task 4: update-task コマンドの実装

**Files:**
- Create: `cmd/clickup/update_task.go`

- [ ] **Step 1: `cmd/clickup/update_task.go` を作成する**

```go
// cmd/clickup/update_task.go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hiraking/click-up-client/internal/client"
	"github.com/hiraking/click-up-client/internal/dateparse"
	"github.com/hiraking/click-up-client/internal/models"
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

func newUpdateTaskCmd() *cobra.Command {
	var name string
	var description string
	var status string
	var priority string
	var dueDateStr string
	var startDateStr string
	var timeEstimateMin int
	var parentID string
	var clearFields []string

	cmd := &cobra.Command{
		Use:   "update-task <taskId>",
		Short: "Update an existing task and output it as JSON",
		Long: `Update an existing ClickUp task by task ID.

Only the flags you specify will be updated. Flags not provided are left unchanged.

Clearing fields:
  Use --clear FIELD to remove a field's value entirely.
  Accepted values: description, status, priority, due-date, start-date, time-estimate

  Note: 'name' cannot be cleared (required field).
        'parent' cannot be cleared (ClickUp API does not support removing parent).

  Examples:
    update-task abc123 --clear due-date
    update-task abc123 --clear due-date --clear priority
    update-task abc123 --name "New Name" --clear description`,
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
				!changed("time-estimate") && !changed("parent") && len(clearFields) == 0 {
				return fmt.Errorf("Error: no fields specified to update.")
			}

			cfg, err := loadConfig()
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
				t, err := dateparse.ParseISO(dueDateStr, "due-date")
				if err != nil {
					return err
				}
				req.DueDate = &t
			}
			if changed("start-date") {
				t, err := dateparse.ParseISO(startDateStr, "start-date")
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
				req.Parent = &parentID
			}

			c := client.New(cfg.APIKey)
			task, err := c.UpdateTask(context.Background(), taskID, req)
			if err != nil {
				return err
			}
			return printJSON(task)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New task name.")
	cmd.Flags().StringVar(&description, "description", "", "New task description.")
	cmd.Flags().StringVar(&status, "status", "", "New status name (e.g. \"to do\", \"in progress\").")
	cmd.Flags().StringVar(&priority, "priority", "", "New priority: urgent, high, normal, or low.")
	cmd.Flags().StringVar(&dueDateStr, "due-date", "", "New due date as ISO 8601. Timezone-less values are treated as JST (+09:00).")
	cmd.Flags().StringVar(&startDateStr, "start-date", "", "New start date as ISO 8601. Timezone-less values are treated as JST (+09:00).")
	cmd.Flags().IntVar(&timeEstimateMin, "time-estimate", 0, "New time estimate in minutes.")
	cmd.Flags().StringVar(&parentID, "parent", "", "New parent task ID.")
	cmd.Flags().StringArrayVar(&clearFields, "clear", nil,
		"Field to clear (repeatable). Accepted: description, status, priority, due-date, start-date, time-estimate.\n"+
			"Use --clear FIELD to remove a field's value from the task.")

	return cmd
}
```

- [ ] **Step 2: コンパイルを確認する**

```
go build ./cmd/clickup/...
```

期待: エラーなし

- [ ] **Step 3: コミットする**

```bash
git add cmd/clickup/update_task.go
git commit -m "feat: add update-task command"
```

---

## Task 5: main.go にコマンド登録

**Files:**
- Modify: `cmd/clickup/main.go`

- [ ] **Step 1: `update-task` コマンドを登録する**

`cmd/clickup/main.go` の `rootCmd.AddCommand(newCreateTaskCmd())` の後ろに以下を追加:

```go
rootCmd.AddCommand(newUpdateTaskCmd())
```

変更後の `main.go`:

```go
// cmd/clickup/main.go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:           "clickup",
		Short:         "ClickUp API CLI wrapper",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(newGetTaskCmd())
	rootCmd.AddCommand(newGetTasksCmd())
	rootCmd.AddCommand(newCreateTaskCmd())
	rootCmd.AddCommand(newUpdateTaskCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: ヘルプが表示されることを確認する**

```
go run ./cmd/clickup/... update-task --help
```

期待: `update-task <taskId>` の Usage、フラグ一覧、Long 説明が表示される

- [ ] **Step 3: 引数なしエラーを確認する**

```
go run ./cmd/clickup/... update-task
```

期待: `Error: accepts 1 arg(s), received 0` のようなエラー

- [ ] **Step 4: フラグなしエラーを確認する**

```
go run ./cmd/clickup/... update-task abc123
```

期待: `Error: no fields specified to update.`

- [ ] **Step 5: 不正な --clear フィールドのエラーを確認する**

```
go run ./cmd/clickup/... update-task abc123 --clear foo
```

期待: `Error: invalid field for --clear: "foo". Accepted: description, status, priority, due-date, start-date, time-estimate`

- [ ] **Step 6: 全テストを実行する**

```
go test ./...
```

期待: 全テスト PASS

- [ ] **Step 7: コミットする**

```bash
git add cmd/clickup/main.go
git commit -m "feat: register update-task command in main"
```

---

## Task 6: README の更新

**Files:**
- Modify: `README.md`

- [ ] **Step 1: README の既存コマンドセクションを確認する**

```
go run ./cmd/clickup/... --help
```

README の構造に合わせて `update-task` セクションを追加する。

- [ ] **Step 2: README に `update-task` を追加する**

既存の `create-task` セクションの後ろに以下を追加（README の既存フォーマットに合わせること）:

````markdown
### update-task

タスクを更新します。指定したフラグのフィールドのみ更新されます。

```bash
clickup update-task <taskId> [flags]
```

**フラグ:**

| フラグ | 説明 |
|---|---|
| `--name` | タスク名 |
| `--description` | 説明 |
| `--status` | ステータス名 |
| `--priority` | 優先度: `urgent` / `high` / `normal` / `low` |
| `--due-date` | 期日（ISO 8601。タイムゾーンなしは JST 扱い）|
| `--start-date` | 開始日（ISO 8601。タイムゾーンなしは JST 扱い）|
| `--time-estimate` | 見積もり時間（分）|
| `--parent` | 親タスク ID |
| `--clear FIELD` | フィールドをクリアする（繰り返し可）|

**`--clear` で指定できるフィールド:**  
`description`, `status`, `priority`, `due-date`, `start-date`, `time-estimate`

> `name` と `parent` はクリア不可。

**例:**

```bash
# 名前を変更する
clickup update-task abc123 --name "新しい名前"

# ステータスと優先度を同時に変更する
clickup update-task abc123 --status "in progress" --priority high

# 期日をクリアする
clickup update-task abc123 --clear due-date

# 名前を変更しつつ説明をクリアする
clickup update-task abc123 --name "新しい名前" --clear description

# 複数フィールドをクリアする
clickup update-task abc123 --clear due-date --clear priority
```

出力: 更新後のタスクを JSON で出力します。
````

- [ ] **Step 3: コミットする**

```bash
git add README.md
git commit -m "docs: add update-task command documentation to README"
```

---

## 完了チェック

- [ ] `go build ./...` がエラーなし
- [ ] `go test ./...` が全テスト PASS
- [ ] `clickup update-task --help` で `--clear` の仕様が明記されている
- [ ] README に `update-task` セクションが追加されている
