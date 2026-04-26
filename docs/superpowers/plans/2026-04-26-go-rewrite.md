# ClickUp CLI — Go Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** C# (.NET 10) の ClickUp CLI を Go (cobra + viper) に完全書き換えし、同一のコマンドインターフェース・JSON出力形式を維持する。

**Architecture:** `internal/` パッケージ群に責務を分割し、`cmd/clickup/` に cobra コマンド定義を置く。Raw API 型は HTTP クライアント実装の詳細として `internal/client/` に内包し、外部に見せるのは `internal/models/` の DTO のみ。ツリー構築は `internal/tree/` が担当し、`[]models.TaskSummary` を入出力とする。

**Tech Stack:** Go 1.23+、cobra v1.9.1、viper v1.20.1、testify v1.10.0、標準ライブラリ (net/http, encoding/json)

---

## ファイル構成

### 新規作成

```
go.mod
go.sum                            ← go mod tidy で自動生成
config.sample.json                ← 既存ファイルをルートに移動
cmd/clickup/
  main.go                         ← エントリポイント、rootCmd
  get_tasks.go                    ← get-tasks コマンド
  get_task.go                     ← get-task コマンド
  create_task.go                  ← create-task コマンド + parsePriority()
  helpers.go                      ← loadConfig(), printJSON()
internal/
  models/
    task_summary.go               ← TaskSummary struct
    create_task.go                ← CreateTaskRequest, TaskPriority
  client/
    client.go                     ← ClickUpClient interface + httpClient 実装
    raw_types.go                  ← 非公開: rawTask, rawTaskStatus 等
    raw_create.go                 ← 非公開: rawCreateTaskBody
    mapper.go                     ← 非公開: toSummary(), mapToRawCreateBody()
    mapper_test.go                ← toSummary() のユニットテスト
  tree/
    builder.go                    ← Build([]TaskSummary) []TaskSummary
    builder_test.go
  config/
    config.go                     ← AppConfig, Load(path string)
  dateparse/
    parse.go                      ← ParseISO(value, optionName string)
    parse_test.go
```

### 削除

- `src/` ディレクトリ全体 (C# ライブラリ + CLI)
- `tests/` ディレクトリ全体 (C# テスト)
- `ClickUpClient.slnx`

### 変更

- `.gitignore` → .NET エントリ削除、Go エントリ追加
- `README.md` → Go 版ビルド手順・コマンドリファレンスに全面更新

---

## Task 1: C# 削除・Go モジュール初期化

**Files:**
- Delete: `src/`, `tests/`, `ClickUpClient.slnx`
- Create: `go.mod`, `.gitignore` (更新), `config.sample.json` (ルートへ移動)

- [ ] **Step 1: C# 成果物を削除**

```powershell
Remove-Item -Recurse -Force src, tests, ClickUpClient.slnx
```

- [ ] **Step 2: go.mod を作成**

```
module github.com/hiraking/click-up-client

go 1.23
```

- [ ] **Step 3: cobra・viper・testify を追加**

```bash
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
go get github.com/stretchr/testify@latest
go mod tidy
```

Expected: `go.sum` が生成される。

- [ ] **Step 4: .gitignore を Go 用に更新**

```
# Go build artifacts
*.exe
*.dll
*.so
*.dylib
out/

# Test cache
*.test
*.out

# CLI runtime config (contains API keys)
config.json

# Local git worktrees
.worktrees/
```

- [ ] **Step 5: config.sample.json をルートに移動**

既存の `src/ClickUpCli/config.sample.json` をリポジトリルートに移動（内容は変更なし）。

```powershell
# config.sample.json は Task 1 完了後にルートに存在するはず。
# 削除前に内容を確認してルートにコピー済みであることを確認する。
```

`config.sample.json` の内容:
```json
{
  "apiKey": "pk_YOUR_API_KEY_HERE",
  "teamId": "YOUR_TEAM_ID_HERE",
  "lists": {
    "my-list": "LIST_ID_HERE"
  }
}
```

- [ ] **Step 6: ディレクトリ構造を作成**

```bash
mkdir -p cmd/clickup internal/models internal/client internal/tree internal/config internal/dateparse
```

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "chore: remove C# project, initialize Go module

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 2: internal/models — DTO 定義

**Files:**
- Create: `internal/models/task_summary.go`
- Create: `internal/models/create_task.go`

- [ ] **Step 1: task_summary.go を作成**

```go
// internal/models/task_summary.go
package models

import "time"

// TaskSummary はエージェント向けの整形済みタスク DTO。
// Subtasks に子タスクをネストして保持するツリーノードを兼ねる。
type TaskSummary struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Status      string        `json:"status"`
	Priority    *string       `json:"priority"`
	ParentID    *string       `json:"parentId"`
	URL         string        `json:"url"`
	DueDate     *time.Time    `json:"dueDate"`
	Description *string       `json:"description"`
	ListID      string        `json:"listId"`
	ListName    string        `json:"listName"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
	Subtasks    []TaskSummary `json:"subtasks"`
}
```

- [ ] **Step 2: create_task.go を作成**

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
}
```

- [ ] **Step 3: ビルド確認**

```bash
go build ./internal/models/...
```

Expected: エラーなし。

- [ ] **Step 4: Commit**

```bash
git add internal/models/
git commit -m "feat: add models package (TaskSummary, CreateTaskRequest)

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 3: internal/client — Raw 型定義

**Files:**
- Create: `internal/client/raw_types.go`
- Create: `internal/client/raw_create.go`

- [ ] **Step 1: raw_types.go を作成**

```go
// internal/client/raw_types.go
package client

// rawTask は GET /v2/team/{teamId}/task および GET /v2/task/{taskId} のレスポンス要素。
type rawTask struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description *string       `json:"description"`
	Status      rawTaskStatus `json:"status"`
	Parent      *string       `json:"parent"`
	Priority    *rawPriority  `json:"priority"`
	DueDate     *string       `json:"due_date"`
	StartDate   *string       `json:"start_date"`
	DateCreated string        `json:"date_created"`
	DateUpdated string        `json:"date_updated"`
	URL         string        `json:"url"`
	List        rawListRef    `json:"list"`
	TeamID      string        `json:"team_id"`
}

type rawTaskStatus struct {
	Status string `json:"status"`
}

type rawPriority struct {
	Priority string `json:"priority"`
}

type rawListRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type rawGetTasksResponse struct {
	Tasks []rawTask `json:"tasks"`
}
```

- [ ] **Step 2: raw_create.go を作成**

```go
// internal/client/raw_create.go
package client

// rawCreateTaskBody は POST /v2/list/{listId}/task のリクエストボディ。
// omitempty で nil フィールドを省略する。
type rawCreateTaskBody struct {
	Name          string  `json:"name"`
	Parent        *string `json:"parent,omitempty"`
	Description   *string `json:"description,omitempty"`
	Status        *string `json:"status,omitempty"`
	Priority      *int    `json:"priority,omitempty"`
	DueDate       *int64  `json:"due_date,omitempty"`
	DueDateTime   *bool   `json:"due_date_time,omitempty"`
	StartDate     *int64  `json:"start_date,omitempty"`
	StartDateTime *bool   `json:"start_date_time,omitempty"`
	TimeEstimate  *int    `json:"time_estimate,omitempty"`
}
```

- [ ] **Step 3: ビルド確認**

```bash
go build ./internal/client/...
```

Expected: エラーなし。

- [ ] **Step 4: Commit**

```bash
git add internal/client/raw_types.go internal/client/raw_create.go
git commit -m "feat: add client raw API types

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 4: internal/client — mapper (TDD)

**Files:**
- Create: `internal/client/mapper_test.go`
- Create: `internal/client/mapper.go`

- [ ] **Step 1: mapper_test.go を作成（失敗するテストを先に書く）**

```go
// internal/client/mapper_test.go
package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSummary_FullFields(t *testing.T) {
	dueDate := "1567780450202"
	parentID := "parent123"
	desc := "task description"
	raw := rawTask{
		ID:          "task123",
		Name:        "Test Task",
		Description: &desc,
		Status:      rawTaskStatus{Status: "in progress"},
		Parent:      &parentID,
		Priority:    &rawPriority{Priority: "high"},
		DueDate:     &dueDate,
		DateCreated: "1567780450000",
		DateUpdated: "1567780451000",
		URL:         "https://app.clickup.com/t/task123",
		List:        rawListRef{ID: "list1", Name: "My List"},
	}

	s := toSummary(raw)

	assert.Equal(t, "task123", s.ID)
	assert.Equal(t, "Test Task", s.Name)
	assert.Equal(t, "in progress", s.Status)
	require.NotNil(t, s.Priority)
	assert.Equal(t, "high", *s.Priority)
	require.NotNil(t, s.ParentID)
	assert.Equal(t, "parent123", *s.ParentID)
	assert.Equal(t, "https://app.clickup.com/t/task123", s.URL)
	require.NotNil(t, s.DueDate)
	assert.Equal(t, int64(1567780450202), s.DueDate.UnixMilli())
	require.NotNil(t, s.Description)
	assert.Equal(t, "task description", *s.Description)
	assert.Equal(t, "list1", s.ListID)
	assert.Equal(t, "My List", s.ListName)
	assert.Equal(t, int64(1567780450000), s.CreatedAt.UnixMilli())
	assert.Equal(t, int64(1567780451000), s.UpdatedAt.UnixMilli())
	assert.NotNil(t, s.Subtasks)
	assert.Empty(t, s.Subtasks)
}

func TestToSummary_NullableFields(t *testing.T) {
	raw := rawTask{
		ID:          "task456",
		Name:        "Minimal Task",
		Status:      rawTaskStatus{Status: "to do"},
		DateCreated: "1567780450000",
		DateUpdated: "1567780450000",
		URL:         "https://app.clickup.com/t/task456",
		List:        rawListRef{ID: "list2", Name: "Work"},
	}

	s := toSummary(raw)

	assert.Nil(t, s.Priority)
	assert.Nil(t, s.ParentID)
	assert.Nil(t, s.DueDate)
	assert.Nil(t, s.Description)
}

func TestMapToRawCreateBody_WithDueDateTime(t *testing.T) {
	dueDate := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	startDate := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	dur := 2 * time.Hour
	p := int(2) // high
	pModel := priorityFromInt(p)

	req := buildCreateRequest(t, dueDate, startDate, dur, pModel)

	body := mapToRawCreateBody(req)

	assert.Equal(t, dueDate.UnixMilli(), *body.DueDate)
	assert.True(t, *body.DueDateTime)   // 18:00 → has time
	assert.False(t, *body.StartDateTime) // 00:00 → no time
	assert.Equal(t, int64(startDate.UnixMilli()), *body.StartDate)
	assert.Equal(t, int(dur.Milliseconds()), *body.TimeEstimate)
	assert.Equal(t, p, *body.Priority)
}

// helpers for tests
func priorityFromInt(n int) models.TaskPriority {
	return models.TaskPriority(n)
}
```

Wait — `mapToRawCreateBody` test imports `models`. Let me adjust the test to use correct imports.

Actually, replace the test above with this corrected version:

```go
// internal/client/mapper_test.go
package client

import (
	"testing"
	"time"

	"github.com/hiraking/click-up-client/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSummary_FullFields(t *testing.T) {
	dueDate := "1567780450202"
	parentID := "parent123"
	desc := "task description"
	raw := rawTask{
		ID:          "task123",
		Name:        "Test Task",
		Description: &desc,
		Status:      rawTaskStatus{Status: "in progress"},
		Parent:      &parentID,
		Priority:    &rawPriority{Priority: "high"},
		DueDate:     &dueDate,
		DateCreated: "1567780450000",
		DateUpdated: "1567780451000",
		URL:         "https://app.clickup.com/t/task123",
		List:        rawListRef{ID: "list1", Name: "My List"},
	}

	s := toSummary(raw)

	assert.Equal(t, "task123", s.ID)
	assert.Equal(t, "Test Task", s.Name)
	assert.Equal(t, "in progress", s.Status)
	require.NotNil(t, s.Priority)
	assert.Equal(t, "high", *s.Priority)
	require.NotNil(t, s.ParentID)
	assert.Equal(t, "parent123", *s.ParentID)
	assert.Equal(t, "https://app.clickup.com/t/task123", s.URL)
	require.NotNil(t, s.DueDate)
	assert.Equal(t, int64(1567780450202), s.DueDate.UnixMilli())
	require.NotNil(t, s.Description)
	assert.Equal(t, "task description", *s.Description)
	assert.Equal(t, "list1", s.ListID)
	assert.Equal(t, "My List", s.ListName)
	assert.Equal(t, int64(1567780450000), s.CreatedAt.UnixMilli())
	assert.Equal(t, int64(1567780451000), s.UpdatedAt.UnixMilli())
	assert.NotNil(t, s.Subtasks)
	assert.Empty(t, s.Subtasks)
}

func TestToSummary_NullableFields(t *testing.T) {
	raw := rawTask{
		ID:          "task456",
		Name:        "Minimal Task",
		Status:      rawTaskStatus{Status: "to do"},
		DateCreated: "1567780450000",
		DateUpdated: "1567780450000",
		URL:         "https://app.clickup.com/t/task456",
		List:        rawListRef{ID: "list2", Name: "Work"},
	}

	s := toSummary(raw)

	assert.Nil(t, s.Priority)
	assert.Nil(t, s.ParentID)
	assert.Nil(t, s.DueDate)
	assert.Nil(t, s.Description)
}

func TestMapToRawCreateBody_DueDateTimeFlag(t *testing.T) {
	// 時刻あり → due_date_time = true
	withTime := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	dur := 2 * time.Hour
	pri := models.PriorityHigh
	req := models.CreateTaskRequest{
		Name:         "Test",
		DueDate:      &withTime,
		TimeEstimate: &dur,
		Priority:     &pri,
	}

	body := mapToRawCreateBody(req)

	assert.Equal(t, withTime.UnixMilli(), *body.DueDate)
	assert.True(t, *body.DueDateTime)
	assert.Equal(t, int(dur.Milliseconds()), *body.TimeEstimate)
	assert.Equal(t, 2, *body.Priority) // PriorityHigh = 2
}

func TestMapToRawCreateBody_MidnightHasNoTime(t *testing.T) {
	// 00:00:00 → due_date_time = false
	midnight := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	req := models.CreateTaskRequest{
		Name:    "Test",
		DueDate: &midnight,
	}

	body := mapToRawCreateBody(req)

	assert.False(t, *body.DueDateTime)
}

func TestMapToRawCreateBody_NilFields(t *testing.T) {
	req := models.CreateTaskRequest{Name: "Minimal"}

	body := mapToRawCreateBody(req)

	assert.Equal(t, "Minimal", body.Name)
	assert.Nil(t, body.Parent)
	assert.Nil(t, body.Description)
	assert.Nil(t, body.Status)
	assert.Nil(t, body.Priority)
	assert.Nil(t, body.DueDate)
	assert.Nil(t, body.StartDate)
	assert.Nil(t, body.TimeEstimate)
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./internal/client/... 2>&1
```

Expected: `undefined: toSummary` でコンパイルエラー。

- [ ] **Step 3: mapper.go を実装**

```go
// internal/client/mapper.go
package client

import (
	"strconv"
	"time"

	"github.com/hiraking/click-up-client/internal/models"
)

// toSummary は rawTask を models.TaskSummary に変換する。
// Subtasks は空スライス（nil でなく []）で初期化する。
func toSummary(raw rawTask) models.TaskSummary {
	return models.TaskSummary{
		ID:          raw.ID,
		Name:        raw.Name,
		Status:      raw.Status.Status,
		Priority:    priorityStr(raw.Priority),
		ParentID:    raw.Parent,
		URL:         raw.URL,
		DueDate:     parseUnixMsPtr(raw.DueDate),
		Description: raw.Description,
		ListID:      raw.List.ID,
		ListName:    raw.List.Name,
		CreatedAt:   parseUnixMs(raw.DateCreated),
		UpdatedAt:   parseUnixMs(raw.DateUpdated),
		Subtasks:    []models.TaskSummary{},
	}
}

// mapToRawCreateBody は models.CreateTaskRequest を rawCreateTaskBody に変換する。
func mapToRawCreateBody(req models.CreateTaskRequest) rawCreateTaskBody {
	body := rawCreateTaskBody{
		Name:        req.Name,
		Parent:      req.ParentID,
		Description: req.Description,
		Status:      req.Status,
	}

	if req.Priority != nil {
		p := int(*req.Priority)
		body.Priority = &p
	}

	if req.DueDate != nil {
		ms := req.DueDate.UnixMilli()
		body.DueDate = &ms
		hasTime := hasTimeComponent(*req.DueDate)
		body.DueDateTime = &hasTime
	}

	if req.StartDate != nil {
		ms := req.StartDate.UnixMilli()
		body.StartDate = &ms
		hasTime := hasTimeComponent(*req.StartDate)
		body.StartDateTime = &hasTime
	}

	if req.TimeEstimate != nil {
		ms := int(req.TimeEstimate.Milliseconds())
		body.TimeEstimate = &ms
	}

	return body
}

func priorityStr(p *rawPriority) *string {
	if p == nil {
		return nil
	}
	s := p.Priority
	return &s
}

func parseUnixMsPtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	ms, err := strconv.ParseInt(*s, 10, 64)
	if err != nil {
		return nil
	}
	t := time.UnixMilli(ms).UTC()
	return &t
}

func parseUnixMs(s string) time.Time {
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

// hasTimeComponent は time.Time の時刻部分（時・分・秒・ナノ秒）が
// すべてゼロかどうかを判定する。
func hasTimeComponent(t time.Time) bool {
	return t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0 || t.Nanosecond() != 0
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./internal/client/... -v
```

Expected: 全テスト PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/client/mapper.go internal/client/mapper_test.go
git commit -m "feat: add client mapper (rawTask <-> TaskSummary)

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 5: internal/tree — ツリー構築 (TDD)

**Files:**
- Create: `internal/tree/builder_test.go`
- Create: `internal/tree/builder.go`

- [ ] **Step 1: builder_test.go を作成**

```go
// internal/tree/builder_test.go
package tree

import (
	"testing"
	"time"

	"github.com/hiraking/click-up-client/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTask(id string, parentID *string) models.TaskSummary {
	return models.TaskSummary{
		ID:        id,
		Name:      "Task " + id,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Subtasks:  []models.TaskSummary{},
	}
}

func strPtr(s string) *string { return &s }

func TestBuild_Empty(t *testing.T) {
	result := Build([]models.TaskSummary{})
	assert.Empty(t, result)
}

func TestBuild_SingleRoot(t *testing.T) {
	tasks := []models.TaskSummary{makeTask("a", nil)}
	result := Build(tasks)

	require.Len(t, result, 1)
	assert.Equal(t, "a", result[0].ID)
	assert.Empty(t, result[0].Subtasks)
}

func TestBuild_ParentChild(t *testing.T) {
	parent := makeTask("parent", nil)
	child := makeTask("child", strPtr("parent"))
	child.ParentID = strPtr("parent")

	result := Build([]models.TaskSummary{parent, child})

	require.Len(t, result, 1)
	assert.Equal(t, "parent", result[0].ID)
	require.Len(t, result[0].Subtasks, 1)
	assert.Equal(t, "child", result[0].Subtasks[0].ID)
}

func TestBuild_MultiLevel(t *testing.T) {
	grandparent := makeTask("gp", nil)
	parent := makeTask("p", strPtr("gp"))
	parent.ParentID = strPtr("gp")
	child := makeTask("c", strPtr("p"))
	child.ParentID = strPtr("p")

	result := Build([]models.TaskSummary{grandparent, parent, child})

	require.Len(t, result, 1)
	assert.Equal(t, "gp", result[0].ID)
	require.Len(t, result[0].Subtasks, 1)
	assert.Equal(t, "p", result[0].Subtasks[0].ID)
	require.Len(t, result[0].Subtasks[0].Subtasks, 1)
	assert.Equal(t, "c", result[0].Subtasks[0].Subtasks[0].ID)
}

func TestBuild_OrphanTreatedAsRoot(t *testing.T) {
	// 親が取得リストに存在しないサブタスク → ルート扱い
	orphan := makeTask("orphan", strPtr("missing-parent"))
	orphan.ParentID = strPtr("missing-parent")

	result := Build([]models.TaskSummary{orphan})

	require.Len(t, result, 1)
	assert.Equal(t, "orphan", result[0].ID)
}

func TestBuild_SortedByID(t *testing.T) {
	t1 := makeTask("z", nil)
	t2 := makeTask("a", nil)
	t3 := makeTask("m", nil)

	result := Build([]models.TaskSummary{t1, t2, t3})

	require.Len(t, result, 3)
	assert.Equal(t, "a", result[0].ID)
	assert.Equal(t, "m", result[1].ID)
	assert.Equal(t, "z", result[2].ID)
}

func TestBuild_SubtasksNotNil(t *testing.T) {
	// Subtasks は nil ではなく空スライスであること（JSON 出力で [] になる）
	task := makeTask("t", nil)
	result := Build([]models.TaskSummary{task})

	require.Len(t, result, 1)
	assert.NotNil(t, result[0].Subtasks)
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./internal/tree/... 2>&1
```

Expected: `undefined: Build` でコンパイルエラー。

- [ ] **Step 3: builder.go を実装**

```go
// internal/tree/builder.go
package tree

import (
	"sort"

	"github.com/hiraking/click-up-client/internal/models"
)

// Build はフラットな TaskSummary リストを親子関係に基づくツリーに変換する。
// parent が nil または親が入力リストに存在しないタスクをルートとして扱う。
// ルートおよびサブタスクは ID 順にソートされる。
func Build(tasks []models.TaskSummary) []models.TaskSummary {
	if len(tasks) == 0 {
		return []models.TaskSummary{}
	}

	// ID → TaskSummary のマップ
	byID := make(map[string]models.TaskSummary, len(tasks))
	for _, t := range tasks {
		byID[t.ID] = t
	}

	// parentID → 子タスクリスト
	children := make(map[string][]models.TaskSummary)
	for _, t := range tasks {
		if t.ParentID == nil {
			continue
		}
		if _, exists := byID[*t.ParentID]; !exists {
			continue // 親が存在しない → ルート扱い（children に追加しない）
		}
		children[*t.ParentID] = append(children[*t.ParentID], t)
	}

	// 各 children リストを ID 順にソート
	for k := range children {
		ch := children[k]
		sort.Slice(ch, func(i, j int) bool { return ch[i].ID < ch[j].ID })
		children[k] = ch
	}

	// 再帰的に Subtasks を埋める
	var buildNode func(t models.TaskSummary) models.TaskSummary
	buildNode = func(t models.TaskSummary) models.TaskSummary {
		ch, ok := children[t.ID]
		if !ok {
			t.Subtasks = []models.TaskSummary{}
			return t
		}
		subtasks := make([]models.TaskSummary, len(ch))
		for i, c := range ch {
			subtasks[i] = buildNode(c)
		}
		t.Subtasks = subtasks
		return t
	}

	// ルートタスクを抽出（parent == nil OR 親が byID に存在しない）
	var roots []models.TaskSummary
	for _, t := range tasks {
		if t.ParentID == nil {
			roots = append(roots, t)
			continue
		}
		if _, exists := byID[*t.ParentID]; !exists {
			roots = append(roots, t)
		}
	}

	sort.Slice(roots, func(i, j int) bool { return roots[i].ID < roots[j].ID })

	result := make([]models.TaskSummary, len(roots))
	for i, r := range roots {
		result[i] = buildNode(r)
	}
	return result
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./internal/tree/... -v
```

Expected: 全テスト PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/tree/
git commit -m "feat: add tree builder

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 6: internal/dateparse — ISO 8601 パース (TDD)

**Files:**
- Create: `internal/dateparse/parse_test.go`
- Create: `internal/dateparse/parse.go`

- [ ] **Step 1: parse_test.go を作成**

```go
// internal/dateparse/parse_test.go
package dateparse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var jstOffset = 9 * 60 * 60 // +09:00 in seconds

func TestParseISO_WithZ(t *testing.T) {
	t0, err := ParseISO("2026-05-01T00:00:00Z", "--due-after")
	require.NoError(t, err)
	assert.Equal(t, int64(0), t0.UTC().Sub(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)).Milliseconds())
	_, offset := t0.Zone()
	assert.Equal(t, 0, offset) // UTC
}

func TestParseISO_WithExplicitPlusOffset(t *testing.T) {
	t0, err := ParseISO("2026-05-01T09:00:00+09:00", "--due-date")
	require.NoError(t, err)
	_, offset := t0.Zone()
	assert.Equal(t, jstOffset, offset)
	assert.Equal(t, 9, t0.Hour())
}

func TestParseISO_NoOffset_TreatedAsJST(t *testing.T) {
	t0, err := ParseISO("2026-05-01T09:00:00", "--due-date")
	require.NoError(t, err)
	_, offset := t0.Zone()
	assert.Equal(t, jstOffset, offset)
	assert.Equal(t, 9, t0.Hour())
}

func TestParseISO_DateOnly_TreatedAsJST(t *testing.T) {
	t0, err := ParseISO("2026-05-01", "--due-before")
	require.NoError(t, err)
	_, offset := t0.Zone()
	assert.Equal(t, jstOffset, offset)
	assert.Equal(t, 0, t0.Hour())
	assert.Equal(t, 1, t0.Day())
}

func TestParseISO_WithNegativeOffset(t *testing.T) {
	t0, err := ParseISO("2026-05-01T00:00:00-05:00", "--due-after")
	require.NoError(t, err)
	_, offset := t0.Zone()
	assert.Equal(t, -5*60*60, offset)
}

func TestParseISO_InvalidValue(t *testing.T) {
	_, err := ParseISO("not-a-date", "--due-after")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--due-after")
	assert.Contains(t, err.Error(), "not-a-date")
}

func TestParseISO_DateTimeNoSeconds(t *testing.T) {
	t0, err := ParseISO("2026-05-01T09:00", "--due-date")
	require.NoError(t, err)
	_, offset := t0.Zone()
	assert.Equal(t, jstOffset, offset)
	assert.Equal(t, 9, t0.Hour())
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./internal/dateparse/... 2>&1
```

Expected: `undefined: ParseISO` でコンパイルエラー。

- [ ] **Step 3: parse.go を実装**

```go
// internal/dateparse/parse.go
package dateparse

import (
	"fmt"
	"strings"
	"time"
)

var jst = time.FixedZone("JST", 9*60*60)

// formats は試行するフォーマット一覧。オフセットなしのフォーマットを末尾に並べる。
var formats = []string{
	time.RFC3339Nano,             // 2006-01-02T15:04:05.999999999Z07:00
	time.RFC3339,                 // 2006-01-02T15:04:05Z07:00
	"2006-01-02T15:04:05",        // オフセットなし
	"2006-01-02T15:04",           // 秒なし
	"2006-01-02",                 // 日付のみ
}

// ParseISO は ISO 8601 文字列を time.Time にパースする。
// タイムゾーンオフセットが含まれない場合は JST (+09:00) として扱う。
func ParseISO(value, optionName string) (time.Time, error) {
	for _, format := range formats {
		t, err := time.Parse(format, value)
		if err != nil {
			continue
		}
		if hasExplicitOffset(value) {
			return t, nil
		}
		// オフセットなし → JST に変換
		return time.Date(t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), jst), nil
	}
	return time.Time{}, fmt.Errorf(
		"Error: '--%s' value '%s' is not a valid ISO 8601 datetime.", optionName, value)
}

// hasExplicitOffset は文字列にタイムゾーンオフセット（Z / +HH:MM / -HH:MM）が
// 含まれているかを判定する。
func hasExplicitOffset(value string) bool {
	trimmed := strings.TrimSpace(value)
	if strings.HasSuffix(strings.ToUpper(trimmed), "Z") {
		return true
	}
	// 位置 10 以降に '+' があれば +HH:MM オフセット
	if len(trimmed) > 10 && strings.ContainsRune(trimmed[10:], '+') {
		return true
	}
	// 位置 10 より後に '-' があれば -HH:MM オフセット
	// （日付部分 "2006-01-02" の最後の '-' は位置 7 なので > 9 で判定）
	if idx := strings.LastIndexByte(trimmed, '-'); idx > 9 {
		return true
	}
	return false
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./internal/dateparse/... -v
```

Expected: 全テスト PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/dateparse/
git commit -m "feat: add dateparse package (ISO 8601 with JST fallback)

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 7: internal/config — 設定ファイル読み込み

**Files:**
- Create: `internal/config/config.go`

- [ ] **Step 1: config.go を作成**

```go
// internal/config/config.go
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// AppConfig は config.json のデシリアライズモデル。
type AppConfig struct {
	ApiKey string            `mapstructure:"apiKey"`
	TeamID string            `mapstructure:"teamId"`
	Lists  map[string]string `mapstructure:"lists"`
}

// Load は指定パスの config.json を読み込み AppConfig を返す。
// ファイルが存在しない場合は分かりやすいエラーメッセージを返す。
func Load(path string) (*AppConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf(
				"config.json not found at '%s'. Copy config.sample.json to config.json and fill in your values.", path)
		}
		return nil, fmt.Errorf("failed to read config.json: %w", err)
	}

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config.json is invalid: %w", err)
	}

	if cfg.ApiKey == "" {
		return nil, fmt.Errorf("config.json: 'apiKey' is required")
	}
	if cfg.TeamID == "" {
		return nil, fmt.Errorf("config.json: 'teamId' is required")
	}

	return &cfg, nil
}
```

- [ ] **Step 2: ビルド確認**

```bash
go build ./internal/config/...
```

Expected: エラーなし。

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add config loader (viper)

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 8: internal/client — HTTP クライアント実装

**Files:**
- Create: `internal/client/client.go`

- [ ] **Step 1: client.go を作成**

```go
// internal/client/client.go
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/hiraking/click-up-client/internal/models"
	"github.com/hiraking/click-up-client/internal/tree"
)

const baseURL = "https://api.clickup.com/api/"

// ClickUpClient は ClickUp API の HTTP クライアントインターフェース。
type ClickUpClient interface {
	GetTasks(ctx context.Context, teamID string, opts GetTasksOptions) ([]models.TaskSummary, error)
	GetTask(ctx context.Context, taskID string) (models.TaskSummary, error)
	CreateTask(ctx context.Context, listID string, req models.CreateTaskRequest) (models.TaskSummary, error)
}

// GetTasksOptions は GetTasks のフィルタオプション。
type GetTasksOptions struct {
	IncludeSubtasks bool
	Page            int
	ListIDs         []string
	Statuses        []string
	DueDateGt       *time.Time
	DueDateLt       *time.Time
}

type httpClient struct {
	base   string
	apiKey string
	http   *http.Client
}

// New は apiKey を使う ClickUpClient を返す。
func New(apiKey string) ClickUpClient {
	return &httpClient{
		base:   baseURL,
		apiKey: apiKey,
		http:   &http.Client{},
	}
}

func (c *httpClient) GetTasks(ctx context.Context, teamID string, opts GetTasksOptions) ([]models.TaskSummary, error) {
	rawURL := c.buildGetTasksURL(teamID, opts)

	var resp rawGetTasksResponse
	if err := c.doGet(ctx, rawURL, &resp); err != nil {
		return nil, err
	}

	summaries := make([]models.TaskSummary, len(resp.Tasks))
	for i, t := range resp.Tasks {
		summaries[i] = toSummary(t)
	}
	return tree.Build(summaries), nil
}

func (c *httpClient) GetTask(ctx context.Context, taskID string) (models.TaskSummary, error) {
	rawURL := c.base + "v2/task/" + taskID

	var raw rawTask
	if err := c.doGet(ctx, rawURL, &raw); err != nil {
		return models.TaskSummary{}, err
	}
	return toSummary(raw), nil
}

func (c *httpClient) CreateTask(ctx context.Context, listID string, req models.CreateTaskRequest) (models.TaskSummary, error) {
	rawURL := c.base + "v2/list/" + listID + "/task"
	body := mapToRawCreateBody(req)

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(bodyBytes))
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

func (c *httpClient) doGet(ctx context.Context, rawURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP Error (%d): %s", resp.StatusCode, string(b))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *httpClient) buildGetTasksURL(teamID string, opts GetTasksOptions) string {
	params := url.Values{}
	if opts.IncludeSubtasks {
		params.Set("subtasks", "true")
	} else {
		params.Set("subtasks", "false")
	}
	params.Set("page", strconv.Itoa(opts.Page))
	for _, id := range opts.ListIDs {
		params.Add("list_ids[]", id)
	}
	for _, s := range opts.Statuses {
		params.Add("statuses[]", s)
	}
	if opts.DueDateGt != nil {
		params.Set("due_date_gt", strconv.FormatInt(opts.DueDateGt.UnixMilli(), 10))
	}
	if opts.DueDateLt != nil {
		params.Set("due_date_lt", strconv.FormatInt(opts.DueDateLt.UnixMilli(), 10))
	}
	return c.base + "v2/team/" + teamID + "/task?" + params.Encode()
}
```

- [ ] **Step 2: ビルド確認**

```bash
go build ./internal/client/...
```

Expected: エラーなし。

- [ ] **Step 3: 全テスト確認**

```bash
go test ./...
```

Expected: 全テスト PASS。

- [ ] **Step 4: Commit**

```bash
git add internal/client/client.go
git commit -m "feat: add ClickUp HTTP client

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 9: cmd/clickup — エントリポイント + helpers + get-task

**Files:**
- Create: `cmd/clickup/main.go`
- Create: `cmd/clickup/helpers.go`
- Create: `cmd/clickup/get_task.go`

- [ ] **Step 1: helpers.go を作成**

```go
// cmd/clickup/helpers.go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/hiraking/click-up-client/internal/config"
)

func loadConfig() (*config.AppConfig, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return config.Load(filepath.Join(filepath.Dir(execPath), "config.json"))
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
```

- [ ] **Step 2: get_task.go を作成**

```go
// cmd/clickup/get_task.go
package main

import (
	"context"

	"github.com/hiraking/click-up-client/internal/client"
	"github.com/spf13/cobra"
)

func newGetTaskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-task <taskId>",
		Short: "Get a single task by ID as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			c := client.New(cfg.ApiKey)
			task, err := c.GetTask(context.Background(), args[0])
			if err != nil {
				return err
			}
			return printJSON(task)
		},
	}
}
```

- [ ] **Step 3: main.go を作成**

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
	// get-tasks と create-task は後続タスクで追加する

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: ビルド確認**

```bash
go build ./cmd/clickup/...
```

Expected: エラーなし。

- [ ] **Step 5: Commit**

```bash
git add cmd/clickup/main.go cmd/clickup/helpers.go cmd/clickup/get_task.go
git commit -m "feat: add CLI entry point and get-task command

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 10: cmd/clickup — get-tasks コマンド

**Files:**
- Create: `cmd/clickup/get_tasks.go`
- Modify: `cmd/clickup/main.go`（AddCommand 追加）

- [ ] **Step 1: get_tasks.go を作成**

```go
// cmd/clickup/get_tasks.go
package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hiraking/click-up-client/internal/client"
	"github.com/hiraking/click-up-client/internal/dateparse"
	"github.com/spf13/cobra"
)

func newGetTasksCmd() *cobra.Command {
	var lists []string
	var statuses []string
	var dueAfterStr string
	var dueBeforeStr string
	var noSubtasks bool

	cmd := &cobra.Command{
		Use:   "get-tasks",
		Short: "Get tasks as a JSON tree",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			// --list 名 → list ID に解決
			var listIDs []string
			if len(lists) > 0 {
				listIDs = make([]string, 0, len(lists))
				for _, name := range lists {
					id, ok := cfg.Lists[name]
					if !ok {
						return fmt.Errorf("Error: Unknown list name '%s'. Available: %s",
							name, availableListNames(cfg.Lists))
					}
					listIDs = append(listIDs, id)
				}
			}

			var dueDateGt, dueDateLt *time.Time
			if dueAfterStr != "" {
				t, err := dateparse.ParseISO(dueAfterStr, "due-after")
				if err != nil {
					return err
				}
				dueDateGt = &t
			}
			if dueBeforeStr != "" {
				t, err := dateparse.ParseISO(dueBeforeStr, "due-before")
				if err != nil {
					return err
				}
				dueDateLt = &t
			}

			c := client.New(cfg.ApiKey)
			tasks, err := c.GetTasks(context.Background(), cfg.TeamID, client.GetTasksOptions{
				IncludeSubtasks: !noSubtasks,
				ListIDs:         listIDs,
				Statuses:        statuses,
				DueDateGt:       dueDateGt,
				DueDateLt:       dueDateLt,
			})
			if err != nil {
				return err
			}
			return printJSON(tasks)
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

	return cmd
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

- [ ] **Step 2: main.go に get-tasks を追加**

`cmd/clickup/main.go` の `rootCmd.AddCommand(newGetTaskCmd())` の次の行に追加:

```go
rootCmd.AddCommand(newGetTasksCmd())
```

- [ ] **Step 3: ビルド確認**

```bash
go build ./cmd/clickup/...
```

Expected: エラーなし。

- [ ] **Step 4: Commit**

```bash
git add cmd/clickup/get_tasks.go cmd/clickup/main.go
git commit -m "feat: add get-tasks command

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 11: cmd/clickup — create-task コマンド

**Files:**
- Create: `cmd/clickup/create_task.go`
- Modify: `cmd/clickup/main.go`（AddCommand 追加）

- [ ] **Step 1: create_task.go を作成**

```go
// cmd/clickup/create_task.go
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hiraking/click-up-client/internal/client"
	"github.com/hiraking/click-up-client/internal/dateparse"
	"github.com/hiraking/click-up-client/internal/models"
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
				t, err := dateparse.ParseISO(dueDateStr, "due-date")
				if err != nil {
					return err
				}
				req.DueDate = &t
			}
			if cmd.Flags().Changed("start-date") {
				t, err := dateparse.ParseISO(startDateStr, "start-date")
				if err != nil {
					return err
				}
				req.StartDate = &t
			}
			if cmd.Flags().Changed("time-estimate") {
				d := time.Duration(timeEstimateMin) * time.Minute
				req.TimeEstimate = &d
			}

			c := client.New(cfg.ApiKey)
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
	cmd.Flags().StringVar(&dueDateStr, "due-date", "", "Due date as ISO 8601. Timezone-less values are treated as JST (+09:00).")
	cmd.Flags().StringVar(&startDateStr, "start-date", "", "Start date as ISO 8601. Timezone-less values are treated as JST (+09:00).")
	cmd.Flags().IntVar(&timeEstimateMin, "time-estimate", 0, "Time estimate in minutes.")

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
```

- [ ] **Step 2: main.go に create-task を追加**

`cmd/clickup/main.go` の `rootCmd.AddCommand(newGetTasksCmd())` の次の行に追加:

```go
rootCmd.AddCommand(newCreateTaskCmd())
```

- [ ] **Step 3: 全ビルド・全テスト確認**

```bash
go build ./...
go test ./...
```

Expected: ビルド成功・全テスト PASS。

- [ ] **Step 4: Commit**

```bash
git add cmd/clickup/create_task.go cmd/clickup/main.go
git commit -m "feat: add create-task command

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 12: README 更新 + .gitignore + 最終確認

**Files:**
- Modify: `README.md`（全面更新）
- Confirm: `.gitignore`（Task 1 で更新済み）

- [ ] **Step 1: README.md を更新**

README.md の内容を Go 版に全面更新する。以下で置き換える:

```markdown
# ClickUp CLI

ClickUp REST API v2 の薄い CLI ラッパー。AI エージェントやスクリプトから ClickUp タスクを JSON で取得・作成するためのツール。

## セットアップ

### 1. ビルド

```bash
go build -o out/clickup ./cmd/clickup
```

### 2. 設定ファイルの作成

バイナリと同じディレクトリに `config.json` を作成する（`config.sample.json` をコピーして編集）。

```json
{
  "apiKey": "pk_YOUR_API_KEY_HERE",
  "teamId": "YOUR_TEAM_ID_HERE",
  "lists": {
    "work":  "LIST_ID_1",
    "study": "LIST_ID_2"
  }
}
```

| フィールド | 説明 |
|---|---|
| `apiKey` | ClickUp の Personal API Token（Settings → Apps → API Token） |
| `teamId` | ワークスペース ID（URL の `/w/{teamId}/` から確認） |
| `lists` | リスト名 → リスト ID のマッピング。`--list` オプションで名前を指定するために使う |

> `config.json` は `.gitignore` で除外済み。コミットされない。

---

## コマンドリファレンス

### `get-tasks` — タスク一覧をツリー形式で取得

```
clickup get-tasks [options]
```

| オプション | 型 | 説明 |
|---|---|---|
| `--list <name>` | string | 取得するリスト名（`config.json` の `lists` キー）。複数指定可（`--list work --list study`）。省略時は全リスト |
| `--status <name>` | string | フィルタするステータス名。複数指定可 |
| `--due-after <ISO8601>` | string | この日時より後の due_date を持つタスクに絞り込む |
| `--due-before <ISO8601>` | string | この日時より前の due_date を持つタスクに絞り込む |
| `--no-subtasks` | flag | サブタスクを取得しない（デフォルト: サブタスクあり） |

**出力:** ルートタスクの JSON 配列。サブタスクは各タスクの `subtasks` フィールドにネスト。

> **日付のタイムゾーンについて:** オフセットなしで渡した場合（例: `"2026-05-01"` や `"2026-05-01T09:00"`）は **JST (+09:00)** として扱われる。オフセットを明示した場合（例: `"2026-05-01T00:00:00Z"` や `"2026-05-01T09:00:00+09:00"`）はその値をそのまま使用する。

#### 使用例

```bash
# 全リストのタスクを取得
clickup get-tasks

# work リストのタスクのみ
clickup get-tasks --list work

# work と study リストを指定
clickup get-tasks --list work --list study

# ステータスでフィルタ
clickup get-tasks --list work --status active

# 今日中に期限が来るタスク
clickup get-tasks --due-before 2026-04-21T23:59:59+09:00

# サブタスクなしで取得
clickup get-tasks --list work --no-subtasks
```

---

### `create-task` — タスクを新規作成

```
clickup create-task <name> --list <name> [options]
```

| 引数/オプション | 型 | 説明 |
|---|---|---|
| `name` | string | タスク名（必須） |
| `--list <name>` | string | 作成先リスト名（必須） |
| `--description <text>` | string | タスクの説明 |
| `--parent <taskId>` | string | 親タスク ID。指定するとサブタスクとして作成 |
| `--status <name>` | string | ステータス名（例: `"to do"`, `"in progress"`） |
| `--priority <value>` | string | 優先度: `urgent` / `high` / `normal` / `low` |
| `--due-date <ISO8601>` | string | 期日 |
| `--start-date <ISO8601>` | string | 開始日 |
| `--time-estimate <分>` | int | 見積もり時間（分単位） |

**出力:** 作成されたタスクの JSON オブジェクト。

#### 使用例

```bash
# 最小構成
clickup create-task "新しいタスク" --list work

# オプション全指定
clickup create-task "設計書を書く" --list work \
  --description "アーキテクチャ設計書の作成" \
  --parent "86exa7yq5" \
  --status "to do" \
  --priority high \
  --due-date "2026-05-01T18:00+09:00" \
  --start-date "2026-04-25T09:00" \
  --time-estimate 120
```

---

### `get-task` — 単一タスクを取得

```
clickup get-task <taskId>
```

#### 使用例

```bash
clickup get-task 86exa7yq5
```

---

## 出力フォーマット

`TaskSummary` の camelCase JSON。

```json
{
  "id": "86exa7yq5",
  "name": "英語学習",
  "status": "active",
  "priority": null,
  "parentId": null,
  "url": "https://app.clickup.com/t/86exa7yq5",
  "dueDate": null,
  "description": "",
  "listId": "901817486451",
  "listName": "学習",
  "createdAt": "2026-04-19T15:09:41.393Z",
  "updatedAt": "2026-04-19T16:05:33.346Z",
  "subtasks": []
}
```

---

## エラーハンドリング

エラーは stderr に出力され、exit code 1 で終了する。

| ケース | メッセージ例 |
|---|---|
| `config.json` が見つからない | `config.json not found at '...'` |
| 不明なリスト名 | `Error: Unknown list name 'foo'. Available: work, study` |
| 日付フォーマット不正 | `Error: '--due-after' value '...' is not a valid ISO 8601 datetime.` |
| 不正な優先度 | `Error: Invalid priority 'foo'. Use urgent, high, normal, or low.` |
| API エラー | `HTTP Error (404): ...` |

---

## 注意事項

- ページネーションは page=0 のみ取得（大量タスクがある場合は絞り込みを使う）
- `--due-after` / `--due-before` フィルタは ClickUp API 側で処理される
- `--list` は複数回指定可能（`--list work --list study`）
```

- [ ] **Step 2: 全テスト最終確認**

```bash
go test ./... -v
```

Expected: 全テスト PASS。

- [ ] **Step 3: バイナリビルド確認**

```bash
go build -o out/clickup ./cmd/clickup
./out/clickup --help
```

Expected:
```
ClickUp API CLI wrapper

Usage:
  clickup [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  create-task Create a new task and output it as JSON
  get-task    Get a single task by ID as JSON
  get-tasks   Get tasks as a JSON tree
  help        Help about any command

Flags:
  -h, --help   help for clickup

Use "clickup [command] --help" for more information about a command.
```

- [ ] **Step 4: Commit**

```bash
git add README.md .gitignore
git commit -m "docs: update README for Go rewrite

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```
