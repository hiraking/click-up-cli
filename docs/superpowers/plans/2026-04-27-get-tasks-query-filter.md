# get-tasks --query フィルタ 実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `get-tasks` コマンドに `--query` フラグを追加し、タスク名・説明文に対してcase-insensitiveな部分一致フィルタリングを行う。

**Architecture:** API取得後のflat listに対してクライアントサイドフィルタリングを適用し、`tree.Build` に渡す前に絞り込む。フィルタロジックは `internal/client/client.go` 内に閉じ込め、`GetTasksOptions.Query` フィールドで制御する。

**Tech Stack:** Go, `strings` 標準ライブラリ, cobra, testify

---

## ファイル変更マップ

| ファイル | 変更種別 | 内容 |
|---|---|---|
| `internal/client/client.go` | 変更 | `GetTasksOptions.Query` 追加、`filterByQuery` 関数追加、`GetTasks` 内で呼び出し |
| `internal/client/client_test.go` | 新規作成 | `filterByQuery` のユニットテスト |
| `cmd/clickup/get_tasks.go` | 変更 | `--query` フラグ追加、`GetTasksOptions.Query` に渡す |

---

### Task 1: `GetTasksOptions.Query` フィールドと `filterByQuery` 関数の追加

**Files:**
- Modify: `internal/client/client.go`
- Create: `internal/client/client_test.go`

- [ ] **Step 1: テストファイルを作成して失敗するテストを書く**

`internal/client/client_test.go` を新規作成:

```go
// internal/client/client_test.go
package client

import (
	"testing"

	"github.com/hiraking/click-up-client/internal/models"
	"github.com/stretchr/testify/assert"
)

func ptr(s string) *string { return &s }

func TestFilterByQuery_EmptyQuery(t *testing.T) {
	tasks := []models.TaskSummary{
		{Name: "Fix bug"},
		{Name: "Write tests"},
	}
	result := filterByQuery(tasks, "")
	assert.Equal(t, tasks, result)
}

func TestFilterByQuery_MatchesName(t *testing.T) {
	tasks := []models.TaskSummary{
		{Name: "Fix bug in login"},
		{Name: "Write documentation"},
	}
	result := filterByQuery(tasks, "bug")
	assert.Len(t, result, 1)
	assert.Equal(t, "Fix bug in login", result[0].Name)
}

func TestFilterByQuery_MatchesDescription(t *testing.T) {
	tasks := []models.TaskSummary{
		{Name: "Task A", Description: ptr("バグ修正が必要")},
		{Name: "Task B", Description: ptr("ドキュメント更新")},
	}
	result := filterByQuery(tasks, "バグ")
	assert.Len(t, result, 1)
	assert.Equal(t, "Task A", result[0].Name)
}

func TestFilterByQuery_CaseInsensitive(t *testing.T) {
	tasks := []models.TaskSummary{
		{Name: "Fix Bug"},
		{Name: "write tests"},
	}
	result := filterByQuery(tasks, "BUG")
	assert.Len(t, result, 1)
	assert.Equal(t, "Fix Bug", result[0].Name)
}

func TestFilterByQuery_NilDescription(t *testing.T) {
	tasks := []models.TaskSummary{
		{Name: "Task with nil desc", Description: nil},
	}
	result := filterByQuery(tasks, "nil")
	assert.Len(t, result, 1)
}

func TestFilterByQuery_NoMatch(t *testing.T) {
	tasks := []models.TaskSummary{
		{Name: "Fix bug"},
		{Name: "Write tests"},
	}
	result := filterByQuery(tasks, "deploy")
	assert.Empty(t, result)
}
```

- [ ] **Step 2: テストを実行して失敗を確認する**

```
go test ./internal/client/...
```

期待結果: `undefined: filterByQuery` というコンパイルエラー

- [ ] **Step 3: `GetTasksOptions.Query` を追加し `filterByQuery` を実装する**

`internal/client/client.go` の `GetTasksOptions` 構造体に `Query string` を追加:

```go
type GetTasksOptions struct {
	IncludeSubtasks bool
	ListIDs         []string
	Statuses        []string
	DueDateGt       *time.Time
	DueDateLt       *time.Time
	Query           string
}
```

同ファイルの末尾（`buildGetTasksURL` の後）に `filterByQuery` 関数を追加:

```go
func filterByQuery(tasks []models.TaskSummary, query string) []models.TaskSummary {
	if query == "" {
		return tasks
	}
	q := strings.ToLower(query)
	result := make([]models.TaskSummary, 0, len(tasks))
	for _, t := range tasks {
		desc := ""
		if t.Description != nil {
			desc = *t.Description
		}
		if strings.Contains(strings.ToLower(t.Name+" "+desc), q) {
			result = append(result, t)
		}
	}
	return result
}
```

`import` ブロックに `"strings"` を追加する。`client.go` の既存 import:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hiraking/click-up-client/internal/models"
	"github.com/hiraking/click-up-client/internal/tree"
)
```

- [ ] **Step 4: テストを実行してパスを確認する**

```
go test ./internal/client/...
```

期待結果: `ok  github.com/hiraking/click-up-client/internal/client`

- [ ] **Step 5: コミットする**

```bash
git add internal/client/client.go internal/client/client_test.go
git commit -m "feat: add filterByQuery to client with Query option"
```

---

### Task 2: `GetTasks` 内でフィルタを適用する

**Files:**
- Modify: `internal/client/client.go`

- [ ] **Step 1: `GetTasks` に `filterByQuery` 呼び出しを追加する**

`client.go` の `GetTasks` メソッド内で、`toSummary` による変換の後、`tree.Build` の前にフィルタを適用する。

変更前:
```go
summaries := make([]models.TaskSummary, len(allRaw))
for i, t := range allRaw {
    summaries[i] = toSummary(t)
}
return tree.Build(summaries), nil
```

変更後:
```go
summaries := make([]models.TaskSummary, len(allRaw))
for i, t := range allRaw {
    summaries[i] = toSummary(t)
}
summaries = filterByQuery(summaries, opts.Query)
return tree.Build(summaries), nil
```

- [ ] **Step 2: 全テストを実行してリグレッションがないことを確認する**

```
go test ./...
```

期待結果: 全テストがパス

- [ ] **Step 3: コミットする**

```bash
git add internal/client/client.go
git commit -m "feat: apply filterByQuery in GetTasks before tree build"
```

---

### Task 3: `get-tasks` コマンドに `--query` フラグを追加する

**Files:**
- Modify: `cmd/clickup/get_tasks.go`

- [ ] **Step 1: `--query` フラグを追加してオプションに渡す**

`cmd/clickup/get_tasks.go` を変更する。

変更前（変数宣言部分）:
```go
var lists []string
var statuses []string
var dueAfterStr string
var dueBeforeStr string
var noSubtasks bool
```

変更後:
```go
var lists []string
var statuses []string
var dueAfterStr string
var dueBeforeStr string
var noSubtasks bool
var query string
```

`GetTasksOptions` に `Query` を追加（変更前）:
```go
c := client.New(cfg.APIKey)
tasks, err := c.GetTasks(context.Background(), cfg.TeamID, client.GetTasksOptions{
    IncludeSubtasks: !noSubtasks,
    ListIDs:         listIDs,
    Statuses:        statuses,
    DueDateGt:       dueDateGt,
    DueDateLt:       dueDateLt,
})
```

変更後:
```go
c := client.New(cfg.APIKey)
tasks, err := c.GetTasks(context.Background(), cfg.TeamID, client.GetTasksOptions{
    IncludeSubtasks: !noSubtasks,
    ListIDs:         listIDs,
    Statuses:        statuses,
    DueDateGt:       dueDateGt,
    DueDateLt:       dueDateLt,
    Query:           query,
})
```

フラグ登録（既存フラグの末尾に追加）:
```go
cmd.Flags().StringVar(&query, "query", "",
    "Case-insensitive substring to match against task name and description.")
```

- [ ] **Step 2: ビルドが通ることを確認する**

```
go build ./...
```

期待結果: エラーなし

- [ ] **Step 3: 手動で動作確認する**

```
go run ./cmd/clickup get-tasks --query "テスト"
```

期待結果: 名前または説明に "テスト" を含むタスクのみがJSON出力される

- [ ] **Step 4: 全テストを実行する**

```
go test ./...
```

期待結果: 全テストがパス

- [ ] **Step 5: コミットする**

```bash
git add cmd/clickup/get_tasks.go
git commit -m "feat: add --query flag to get-tasks command"
```
