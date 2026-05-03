# time-report コマンド実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ClickUp time entries を指定期間で集計し、List → Top-level task → Breakdown のツリー構造で JSON レポートを出力する `time-report` コマンドを追加する。

**Architecture:** HTTP クライアントに 429 リトライ処理を追加（全 API 共通）。`internal/timereport/` パッケージで集計ロジック（dedup・clip・parent chain 解決・階層構築）を実装。CLI コマンドはクライアントとビルダーを繋ぐだけのシン層とする。

**Tech Stack:** Go 標準ライブラリ、cobra、testify（既存と同じ）

---

## ファイルマップ

| 操作 | パス | 責務 |
|---|---|---|
| 修正 | `internal/client/client.go` | 429 リトライ共通処理・ErrNotFound・GetTimeEntries |
| 新規 | `internal/client/raw_time_entry.go` | rawTimeEntry 型・toTimeEntry マッパー |
| 新規 | `internal/models/time_report.go` | TimeEntry / TimeReport / Row / Hierarchy DTO |
| 新規 | `internal/timereport/builder.go` | 集計ロジック（Build 関数） |
| 新規 | `internal/timereport/builder_test.go` | builder のユニットテスト |
| 新規 | `cmd/clickup/time_report.go` | time-report コマンド実装 |
| 修正 | `cmd/clickup/main.go` | time-report コマンド登録 |
| 修正 | `README.md` | time-report コマンドドキュメント追加 |

---

## Task 1: HTTP クライアントへの 429 リトライ追加

**Files:**
- Modify: `internal/client/client.go`

- [ ] **Step 1: `client.go` に `doWithRetry` と `calcRetryWait` を追加し、既存の `doGet`・`CreateTask`・`UpdateTask` をリファクタリング**

`internal/client/client.go` を以下の完全な内容に置き換える：

```go
// internal/client/client.go
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hiraking/click-up-cli/internal/models"
	"github.com/hiraking/click-up-cli/internal/tree"
)

const baseURL = "https://api.clickup.com/api/"
const maxPages = 10
const maxRetries = 3
const defaultRetryWait = 60 * time.Second

// ErrNotFound is returned when a resource is not found (HTTP 404).
var ErrNotFound = errors.New("not found")

// ClickUpClient は ClickUp API の HTTP クライアントインターフェース。
type ClickUpClient interface {
	GetTasks(ctx context.Context, teamID string, opts GetTasksOptions) ([]models.TaskSummary, error)
	GetTask(ctx context.Context, taskID string) (models.TaskSummary, error)
	CreateTask(ctx context.Context, listID string, req models.CreateTaskRequest) (models.TaskSummary, error)
	UpdateTask(ctx context.Context, taskID string, req models.UpdateTaskRequest) (models.TaskSummary, error)
	GetTimeEntries(ctx context.Context, teamID string, opts GetTimeEntriesOptions) ([]models.TimeEntry, error)
}

// GetTasksOptions は GetTasks のフィルタオプション。
type GetTasksOptions struct {
	IncludeSubtasks bool
	ListIDs         []string
	Statuses        []string
	DueDateGt       *time.Time
	DueDateLt       *time.Time
	Query           string
}

// GetTimeEntriesOptions は GetTimeEntries のオプション。
type GetTimeEntriesOptions struct {
	Start time.Time
	End   time.Time
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
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *httpClient) GetTasks(ctx context.Context, teamID string, opts GetTasksOptions) ([]models.TaskSummary, error) {
	var allRaw []rawTask
	for page := 0; page < maxPages; page++ {
		rawURL := c.buildGetTasksURL(teamID, opts, page)

		var resp rawGetTasksResponse
		if err := c.doGet(ctx, rawURL, &resp); err != nil {
			return nil, err
		}
		allRaw = append(allRaw, resp.Tasks...)
		if resp.LastPage {
			break
		}
		if page == maxPages-1 {
			fmt.Fprintf(os.Stderr, "warning: reached max page limit (%d pages, %d tasks). There may be more tasks. Use filters to narrow down results.\n", maxPages, len(allRaw))
		}
	}

	summaries := make([]models.TaskSummary, len(allRaw))
	for i, t := range allRaw {
		summaries[i] = toSummary(t)
	}
	summaries = filterByQuery(summaries, opts.Query)
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

	respBody, status, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Authorization", c.apiKey)
		httpReq.Header.Set("Content-Type", "application/json")
		return httpReq, nil
	})
	if err != nil {
		return models.TaskSummary{}, err
	}
	if status >= 400 {
		return models.TaskSummary{}, fmt.Errorf("HTTP Error (%d): %s", status, string(respBody))
	}

	var raw rawTask
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to decode response: %w", err)
	}
	return toSummary(raw), nil
}

func (c *httpClient) UpdateTask(ctx context.Context, taskID string, req models.UpdateTaskRequest) (models.TaskSummary, error) {
	rawURL := c.base + "v2/task/" + taskID
	body := mapToRawUpdateBody(req)

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	respBody, status, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, rawURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Authorization", c.apiKey)
		httpReq.Header.Set("Content-Type", "application/json")
		return httpReq, nil
	})
	if err != nil {
		return models.TaskSummary{}, err
	}
	if status >= 400 {
		return models.TaskSummary{}, fmt.Errorf("HTTP Error (%d): %s", status, string(respBody))
	}

	var raw rawTask
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to decode response: %w", err)
	}
	return toSummary(raw), nil
}

func (c *httpClient) GetTimeEntries(ctx context.Context, teamID string, opts GetTimeEntriesOptions) ([]models.TimeEntry, error) {
	const fetchBuffer = 3 * time.Hour
	fetchStart := opts.Start.Add(-fetchBuffer)
	fetchEnd := opts.End.Add(fetchBuffer)

	params := url.Values{}
	params.Set("start_date", strconv.FormatInt(fetchStart.UnixMilli(), 10))
	params.Set("end_date", strconv.FormatInt(fetchEnd.UnixMilli(), 10))
	params.Set("include_location_names", "true")

	rawURL := c.base + "v2/team/" + teamID + "/time_entries?" + params.Encode()

	var resp rawGetTimeEntriesResponse
	if err := c.doGet(ctx, rawURL, &resp); err != nil {
		return nil, err
	}

	entries := make([]models.TimeEntry, len(resp.Data))
	for i, raw := range resp.Data {
		entries[i] = toTimeEntry(raw)
	}
	return entries, nil
}

// doWithRetry は HTTP リクエストを実行し、429 の場合はリトライする。
// 成功時はレスポンスボディと HTTP ステータスコードを返す。
func (c *httpClient) doWithRetry(ctx context.Context, makeReq func() (*http.Request, error)) ([]byte, int, error) {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := makeReq()
		if err != nil {
			return nil, 0, err
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, 0, err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			wait := calcRetryWait(resp)
			resp.Body.Close()
			if attempt == maxRetries {
				return nil, http.StatusTooManyRequests, fmt.Errorf("HTTP Error (429): rate limit exceeded after %d retries", maxRetries)
			}
			fmt.Fprintf(os.Stderr, "warning: rate limited, retrying in %s (attempt %d/%d)...\n", wait.Round(time.Second), attempt+1, maxRetries)
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(wait):
			}
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, resp.StatusCode, readErr
		}
		return body, resp.StatusCode, nil
	}
	return nil, 0, fmt.Errorf("unexpected exit from retry loop")
}

// calcRetryWait は 429 レスポンスから待機時間を算出する。
// X-RateLimit-Reset ヘッダー（Unix 秒）があればリセット時刻まで待機。なければ固定 60 秒。
func calcRetryWait(resp *http.Response) time.Duration {
	resetStr := resp.Header.Get("X-RateLimit-Reset")
	if resetStr != "" {
		resetUnix, err := strconv.ParseInt(resetStr, 10, 64)
		if err == nil {
			wait := time.Until(time.Unix(resetUnix, 0))
			if wait < time.Second {
				wait = time.Second
			}
			return wait
		}
	}
	return defaultRetryWait
}

func (c *httpClient) doGet(ctx context.Context, rawURL string, out any) error {
	body, status, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
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
		return fmt.Errorf("HTTP Error (%d): %s", status, string(body))
	}
	return json.Unmarshal(body, out)
}

func filterByQuery(tasks []models.TaskSummary, query string) []models.TaskSummary {
	if query == "" {
		return tasks
	}
	q := strings.ToLower(query)
	result := make([]models.TaskSummary, 0, len(tasks))
	for _, t := range tasks {
		if strings.Contains(strings.ToLower(t.Name), q) {
			result = append(result, t)
			continue
		}
		if t.Description != nil && strings.Contains(strings.ToLower(*t.Description), q) {
			result = append(result, t)
		}
	}
	return result
}

func (c *httpClient) buildGetTasksURL(teamID string, opts GetTasksOptions, page int) string {
	params := url.Values{}
	if opts.IncludeSubtasks {
		params.Set("subtasks", "true")
	} else {
		params.Set("subtasks", "false")
	}
	params.Set("page", strconv.Itoa(page))
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

- [ ] **Step 2: 既存テストが通ることを確認**

```bash
go test ./internal/client/... ./internal/tree/... ./internal/dateparse/...
```

Expected: `ok github.com/hiraking/click-up-cli/internal/client` など PASS

- [ ] **Step 3: ビルドが通ることを確認**

```bash
go build ./...
```

Expected: エラーなし

- [ ] **Step 4: コミット**

```bash
git add internal/client/client.go
git commit -m "feat: add 429 retry with X-RateLimit-Reset and ErrNotFound to HTTP client

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 2: TimeEntry + TimeReport モデル定義

**Files:**
- Create: `internal/models/time_report.go`

- [ ] **Step 1: `internal/models/time_report.go` を作成**

```go
// internal/models/time_report.go
package models

import "time"

// TimeEntry は ClickUp time entry の処理済み DTO。
type TimeEntry struct {
	ID         string
	TaskID     string
	TaskName   string
	UserID     string
	UserName   string
	Start      time.Time
	End        time.Time
	DurationMs int64  // 元の duration（ms）。負値 = running timer
	// task_location から取得したリスト情報（フォールバック用）
	ListID   string
	ListName string
}

// TimeReport は time-report コマンドの出力 DTO。
type TimeReport struct {
	SchemaVersion int               `json:"schemaVersion"`
	GeneratedAt   time.Time         `json:"generatedAt"`
	Period        TimePeriod        `json:"period"`
	Summary       TimeReportSummary `json:"summary"`
	Hierarchy     []TimeReportList  `json:"hierarchy"`
	Rows          []TimeReportRow   `json:"rows,omitempty"`
}

// TimePeriod は集計期間のメタデータ。
type TimePeriod struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Timezone string    `json:"timezone"`
}

// TimeReportSummary は集計サマリー。
type TimeReportSummary struct {
	TotalDurationMin   int64 `json:"totalDurationMin"`   // 分単位（切り捨て）
	ListCount          int   `json:"listCount"`
	TopLevelTaskCount  int   `json:"topLevelTaskCount"`
	BreakdownTaskCount int   `json:"breakdownTaskCount"`
}

// TimeReportList は List 単位の集計。
type TimeReportList struct {
	ListID      string           `json:"listId"`
	ListName    string           `json:"listName"`
	DurationMin int64            `json:"durationMin"`        // 分単位（切り捨て）
	Tasks       []TimeReportTask `json:"tasks"`
}

// TimeReportTask は top-level task 単位の集計。
type TimeReportTask struct {
	TaskID      string                `json:"taskId"`
	TaskName    string                `json:"taskName"`
	DurationMin int64                 `json:"durationMin"`    // 分単位（切り捨て）
	Breakdown   []TimeReportBreakdown `json:"breakdown"`
}

// TimeReportBreakdown は recorded task 単位の内訳。
// Breakdown は常にフラット（1段）: recorded task = 実際に time entry が記録されたタスク。
// タスク階層が何段あっても top-level task の直下に recorded task が並ぶ。
type TimeReportBreakdown struct {
	TaskID      string `json:"taskId"`
	TaskName    string `json:"taskName"`
	DurationMin int64  `json:"durationMin"`                  // 分単位（切り捨て）
}

// TimeReportRow は後続分析用の正規化済み明細行。
type TimeReportRow struct {
	TimeEntryID        string    `json:"timeEntryId"`
	UserID             string    `json:"userId"`
	UserName           string    `json:"userName"`
	ListID             string    `json:"listId"`
	ListName           string    `json:"listName"`
	TopLevelTaskID     string    `json:"topLevelTaskId"`
	TopLevelTaskName   string    `json:"topLevelTaskName"`
	RecordedTaskID     string    `json:"recordedTaskId"`
	RecordedTaskName   string    `json:"recordedTaskName"`
	OriginalStart      time.Time `json:"originalStart"`
	OriginalEnd        time.Time `json:"originalEnd"`
	OriginalDurationMs int64     `json:"originalDurationMs"`
	ClippedStart       time.Time `json:"clippedStart"`
	ClippedEnd         time.Time `json:"clippedEnd"`
	ClippedDurationMs  int64     `json:"clippedDurationMs"`
}
```

- [ ] **Step 2: ビルドが通ることを確認**

```bash
go build ./...
```

Expected: エラーなし

- [ ] **Step 3: コミット**

```bash
git add internal/models/time_report.go
git commit -m "feat: add TimeEntry and TimeReport models

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 3: rawTimeEntry 型と toTimeEntry マッパー

**Files:**
- Create: `internal/client/raw_time_entry.go`

- [ ] **Step 1: `internal/client/raw_time_entry.go` を作成**

```go
// internal/client/raw_time_entry.go
package client

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/hiraking/click-up-cli/internal/models"
)

// rawTimeEntry は GET /v2/team/{teamId}/time_entries のレスポンス要素。
type rawTimeEntry struct {
	ID           string               `json:"id"`
	Task         *rawEntryTask        `json:"task"`
	User         rawEntryUser         `json:"user"`
	Start        json.Number          `json:"start"`    // Unix ms（API は integer または string を返す）
	End          json.Number          `json:"end"`      // Unix ms
	Duration     string               `json:"duration"` // ms 文字列。負値 = running timer
	TaskLocation rawTimeEntryLocation `json:"task_location"`
}

type rawEntryTask struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type rawEntryUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type rawTimeEntryLocation struct {
	ListID   json.Number `json:"list_id"`   // API が integer または string を返す
	ListName string      `json:"list_name"`
}

type rawGetTimeEntriesResponse struct {
	Data []rawTimeEntry `json:"data"`
}

// toTimeEntry は rawTimeEntry を models.TimeEntry に変換する。
func toTimeEntry(raw rawTimeEntry) models.TimeEntry {
	startMs, _ := strconv.ParseInt(raw.Start.String(), 10, 64)
	endMs, _ := strconv.ParseInt(raw.End.String(), 10, 64)
	durMs, _ := strconv.ParseInt(raw.Duration, 10, 64)

	taskID := ""
	taskName := ""
	if raw.Task != nil {
		taskID = raw.Task.ID
		taskName = raw.Task.Name
	}

	return models.TimeEntry{
		ID:         raw.ID,
		TaskID:     taskID,
		TaskName:   taskName,
		UserID:     strconv.Itoa(raw.User.ID),
		UserName:   raw.User.Username,
		Start:      time.UnixMilli(startMs).UTC(),
		End:        time.UnixMilli(endMs).UTC(),
		DurationMs: durMs,
		ListID:     raw.TaskLocation.ListID.String(),
		ListName:   raw.TaskLocation.ListName,
	}
}
```

- [ ] **Step 2: 既存テストとビルドが通ることを確認**

```bash
go test ./internal/client/... && go build ./...
```

Expected: PASS・エラーなし

- [ ] **Step 3: コミット**

```bash
git add internal/client/raw_time_entry.go
git commit -m "feat: add rawTimeEntry type and toTimeEntry mapper

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 4: timereport ビルダー（TDD）

**Files:**
- Create: `internal/timereport/builder_test.go`
- Create: `internal/timereport/builder.go`

- [ ] **Step 1: `internal/timereport/builder_test.go` を作成（テストを先に書く）**

```go
// internal/timereport/builder_test.go
package timereport_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hiraking/click-up-cli/internal/models"
	"github.com/hiraking/click-up-cli/internal/timereport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var jst = time.FixedZone("JST", 9*60*60)

var (
	reportStart = time.Date(2026, 4, 27, 0, 0, 0, 0, jst)
	reportEnd   = time.Date(2026, 5, 4, 0, 0, 0, 0, jst)
)

func strPtr(s string) *string { return &s }

func makeEntry(id, taskID string, start, end time.Time, durMs int64) models.TimeEntry {
	return models.TimeEntry{
		ID:         id,
		TaskID:     taskID,
		TaskName:   "Task " + taskID,
		UserID:     "user1",
		UserName:   "Test User",
		Start:      start,
		End:        end,
		DurationMs: durMs,
	}
}

func makeTask(id string, parentID *string, listID, listName string) models.TaskSummary {
	return models.TaskSummary{
		ID:       id,
		Name:     "Task " + id,
		ParentID: parentID,
		ListID:   listID,
		ListName: listName,
		Subtasks: []models.TaskSummary{},
	}
}

func noFetch(_ context.Context, id string) (models.TaskSummary, error) {
	return models.TaskSummary{}, errors.New("unexpected fetch for " + id)
}

func mapFetch(tasks map[string]models.TaskSummary) timereport.TaskFetcher {
	return func(_ context.Context, id string) (models.TaskSummary, error) {
		t, ok := tasks[id]
		if !ok {
			return models.TaskSummary{}, errors.New("not found: " + id)
		}
		return t, nil
	}
}

func TestBuild_EmptyEntries(t *testing.T) {
	report, err := timereport.Build(context.Background(), nil, reportStart, reportEnd, noFetch)
	require.NoError(t, err)
	assert.Equal(t, 1, report.SchemaVersion)
	assert.Empty(t, report.Hierarchy)
	assert.Equal(t, int64(0), report.Summary.TotalDurationMin)
	assert.Equal(t, 0, report.Summary.ListCount)
}

func TestBuild_RunningTimerExcluded(t *testing.T) {
	running := makeEntry("e1", "t1",
		time.Date(2026, 4, 28, 9, 0, 0, 0, jst),
		time.Date(2026, 4, 28, 10, 0, 0, 0, jst),
		-1, // negative = running timer
	)
	report, err := timereport.Build(context.Background(), []models.TimeEntry{running}, reportStart, reportEnd, noFetch)
	require.NoError(t, err)
	assert.Empty(t, report.Hierarchy)
	assert.Equal(t, int64(0), report.Summary.TotalDurationMin)
}

func TestBuild_EntryFullyOutsideRange(t *testing.T) {
	outside := makeEntry("e1", "t1",
		time.Date(2026, 5, 5, 9, 0, 0, 0, jst), // reportEnd より後
		time.Date(2026, 5, 5, 10, 0, 0, 0, jst),
		3600000,
	)
	report, err := timereport.Build(context.Background(), []models.TimeEntry{outside}, reportStart, reportEnd, noFetch)
	require.NoError(t, err)
	assert.Empty(t, report.Hierarchy)
	assert.Equal(t, int64(0), report.Summary.TotalDurationMin)
}

func TestBuild_EntryClippedAtStart(t *testing.T) {
	// reportStart の 2h 前から始まり、1h 後に終わる → clipped = 1h = 3600000ms
	entry := makeEntry("e1", "t1",
		time.Date(2026, 4, 26, 22, 0, 0, 0, jst),
		time.Date(2026, 4, 27, 1, 0, 0, 0, jst),
		10800000,
	)
	tasks := map[string]models.TaskSummary{
		"t1": makeTask("t1", nil, "list1", "My List"),
	}
	report, err := timereport.Build(context.Background(), []models.TimeEntry{entry}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	require.Len(t, report.Hierarchy, 1)
	assert.Equal(t, int64(60), report.Summary.TotalDurationMin)
	assert.Equal(t, int64(60), report.Hierarchy[0].DurationMin)

	row := report.Rows[0]
	assert.Equal(t, int64(10800000), row.OriginalDurationMs)
	assert.Equal(t, int64(3600000), row.ClippedDurationMs)
	assert.Equal(t, reportStart, row.ClippedStart)
}

func TestBuild_EntryClippedAtEnd(t *testing.T) {
	// reportEnd の 1h 前から始まり、2h 後に終わる → clipped = 1h = 3600000ms
	entry := makeEntry("e1", "t1",
		time.Date(2026, 5, 3, 23, 0, 0, 0, jst),
		time.Date(2026, 5, 4, 2, 0, 0, 0, jst),
		10800000,
	)
	tasks := map[string]models.TaskSummary{
		"t1": makeTask("t1", nil, "list1", "My List"),
	}
	report, err := timereport.Build(context.Background(), []models.TimeEntry{entry}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	assert.Equal(t, int64(60), report.Summary.TotalDurationMin)
	row := report.Rows[0]
	assert.Equal(t, reportEnd, row.ClippedEnd)
}

func TestBuild_DuplicateEntriesDeduped(t *testing.T) {
	e1 := makeEntry("e1", "t1",
		time.Date(2026, 4, 28, 9, 0, 0, 0, jst),
		time.Date(2026, 4, 28, 10, 0, 0, 0, jst),
		3600000,
	)
	tasks := map[string]models.TaskSummary{
		"t1": makeTask("t1", nil, "list1", "My List"),
	}
	// 同一エントリを2回渡す → 1回分だけ集計
	report, err := timereport.Build(context.Background(), []models.TimeEntry{e1, e1}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)
	assert.Equal(t, int64(60), report.Summary.TotalDurationMin)
	assert.Len(t, report.Rows, 1)
}

func TestBuild_TopLevelTask_BreakdownIsSelf(t *testing.T) {
	// top-level task に直接 time entry が記録された場合、breakdown はその task 自身
	entry := makeEntry("e1", "t1",
		time.Date(2026, 4, 28, 9, 0, 0, 0, jst),
		time.Date(2026, 4, 28, 10, 0, 0, 0, jst),
		3600000,
	)
	tasks := map[string]models.TaskSummary{
		"t1": makeTask("t1", nil, "list1", "My List"),
	}
	report, err := timereport.Build(context.Background(), []models.TimeEntry{entry}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	require.Len(t, report.Hierarchy, 1)
	list := report.Hierarchy[0]
	assert.Equal(t, "list1", list.ListID)
	assert.Equal(t, "My List", list.ListName)
	assert.Equal(t, int64(60), list.DurationMin)

	require.Len(t, list.Tasks, 1)
	task := list.Tasks[0]
	assert.Equal(t, "t1", task.TaskID)
	assert.Equal(t, int64(60), task.DurationMin)

	// top-level task に直接記録 → breakdown は自分自身
	require.Len(t, task.Breakdown, 1)
	assert.Equal(t, "t1", task.Breakdown[0].TaskID)
	assert.Equal(t, int64(60), task.Breakdown[0].DurationMin)
}

func TestBuild_SubtaskResolvesToTopLevel(t *testing.T) {
	// sub1 (parent=top1) に記録 → top1 に集約、breakdown は sub1
	entry := makeEntry("e1", "sub1",
		time.Date(2026, 4, 28, 9, 0, 0, 0, jst),
		time.Date(2026, 4, 28, 11, 0, 0, 0, jst),
		7200000,
	)
	tasks := map[string]models.TaskSummary{
		"sub1": makeTask("sub1", strPtr("top1"), "", ""),
		"top1": makeTask("top1", nil, "list1", "Work"),
	}
	report, err := timereport.Build(context.Background(), []models.TimeEntry{entry}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	require.Len(t, report.Hierarchy, 1)
	assert.Equal(t, "list1", report.Hierarchy[0].ListID)

	require.Len(t, report.Hierarchy[0].Tasks, 1)
	task := report.Hierarchy[0].Tasks[0]
	assert.Equal(t, "top1", task.TaskID)
	assert.Equal(t, int64(120), task.DurationMin)

	require.Len(t, task.Breakdown, 1)
	assert.Equal(t, "sub1", task.Breakdown[0].TaskID)
}

func TestBuild_MultiLevelSubtask_CollapsesToTopLevel(t *testing.T) {
	// A -> B -> C -> D（4段）、D に記録 → A が top-level、breakdown は D（B・C は出ない）
	entry := makeEntry("e1", "D",
		time.Date(2026, 4, 28, 9, 0, 0, 0, jst),
		time.Date(2026, 4, 28, 10, 0, 0, 0, jst),
		3600000,
	)
	tasks := map[string]models.TaskSummary{
		"D": makeTask("D", strPtr("C"), "", ""),
		"C": makeTask("C", strPtr("B"), "", ""),
		"B": makeTask("B", strPtr("A"), "", ""),
		"A": makeTask("A", nil, "list1", "Project"),
	}
	report, err := timereport.Build(context.Background(), []models.TimeEntry{entry}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	require.Len(t, report.Hierarchy, 1)
	require.Len(t, report.Hierarchy[0].Tasks, 1)
	task := report.Hierarchy[0].Tasks[0]
	assert.Equal(t, "A", task.TaskID)

	require.Len(t, task.Breakdown, 1)
	assert.Equal(t, "D", task.Breakdown[0].TaskID)

	assert.Equal(t, 1, report.Summary.TopLevelTaskCount)
	assert.Equal(t, 1, report.Summary.BreakdownTaskCount)
}

func TestBuild_MultipleEntries_Summary(t *testing.T) {
	entries := []models.TimeEntry{
		makeEntry("e1", "t1", time.Date(2026, 4, 28, 9, 0, 0, 0, jst), time.Date(2026, 4, 28, 10, 0, 0, 0, jst), 3600000),
		makeEntry("e2", "t2", time.Date(2026, 4, 29, 9, 0, 0, 0, jst), time.Date(2026, 4, 29, 11, 0, 0, 0, jst), 7200000),
	}
	tasks := map[string]models.TaskSummary{
		"t1": makeTask("t1", nil, "list1", "List 1"),
		"t2": makeTask("t2", nil, "list2", "List 2"),
	}
	report, err := timereport.Build(context.Background(), entries, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	assert.Equal(t, int64(180), report.Summary.TotalDurationMin)
	assert.Equal(t, 2, report.Summary.ListCount)
	assert.Equal(t, 2, report.Summary.TopLevelTaskCount)
	assert.Equal(t, 2, report.Summary.BreakdownTaskCount)
}

func TestBuild_TaskCachePreventsDoubleFetch(t *testing.T) {
	// 同じ subtask を持つ 2 つの time entry → sub1 と top1 はそれぞれ 1 回だけ fetch
	entries := []models.TimeEntry{
		makeEntry("e1", "sub1", time.Date(2026, 4, 28, 9, 0, 0, 0, jst), time.Date(2026, 4, 28, 10, 0, 0, 0, jst), 3600000),
		makeEntry("e2", "sub1", time.Date(2026, 4, 28, 11, 0, 0, 0, jst), time.Date(2026, 4, 28, 12, 0, 0, 0, jst), 3600000),
	}
	tasks := map[string]models.TaskSummary{
		"sub1": makeTask("sub1", strPtr("top1"), "", ""),
		"top1": makeTask("top1", nil, "list1", "Work"),
	}
	fetchCount := 0
	fetch := func(_ context.Context, id string) (models.TaskSummary, error) {
		fetchCount++
		t, ok := tasks[id]
		if !ok {
			return models.TaskSummary{}, errors.New("not found: " + id)
		}
		return t, nil
	}
	report, err := timereport.Build(context.Background(), entries, reportStart, reportEnd, fetch)
	require.NoError(t, err)

	// sub1 と top1 をそれぞれ 1 回ずつ = 計 2 回
	assert.Equal(t, 2, fetchCount)
	assert.Equal(t, int64(120), report.Summary.TotalDurationMin)
}

func TestBuild_ListFallback_UsesEntryList(t *testing.T) {
	// top-level task に ListID がない場合は entry.ListID にフォールバック
	entry := makeEntry("e1", "t1",
		time.Date(2026, 4, 28, 9, 0, 0, 0, jst),
		time.Date(2026, 4, 28, 10, 0, 0, 0, jst),
		3600000,
	)
	entry.ListID = "fallback_list"
	entry.ListName = "Fallback"
	tasks := map[string]models.TaskSummary{
		"t1": makeTask("t1", nil, "", ""), // ListID が空
	}
	report, err := timereport.Build(context.Background(), []models.TimeEntry{entry}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	require.Len(t, report.Hierarchy, 1)
	assert.Equal(t, "fallback_list", report.Hierarchy[0].ListID)
	assert.Equal(t, "Fallback", report.Hierarchy[0].ListName)
}

func TestBuild_Rows_ContainClippedData(t *testing.T) {
	entry := makeEntry("e1", "t1",
		time.Date(2026, 4, 26, 22, 0, 0, 0, jst), // 2h before reportStart
		time.Date(2026, 4, 27, 1, 0, 0, 0, jst),  // 1h after reportStart
		10800000,
	)
	tasks := map[string]models.TaskSummary{
		"t1": makeTask("t1", nil, "list1", "My List"),
	}
	report, err := timereport.Build(context.Background(), []models.TimeEntry{entry}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	require.Len(t, report.Rows, 1)
	row := report.Rows[0]
	assert.Equal(t, "e1", row.TimeEntryID)
	assert.Equal(t, "user1", row.UserID)
	assert.Equal(t, "t1", row.TopLevelTaskID)
	assert.Equal(t, "t1", row.RecordedTaskID)
	assert.Equal(t, int64(10800000), row.OriginalDurationMs)
	assert.Equal(t, int64(3600000), row.ClippedDurationMs)
	assert.Equal(t, reportStart, row.ClippedStart)
	assert.Equal(t, entry.End, row.ClippedEnd)
}

func TestBuild_PeriodAndSchemaVersion(t *testing.T) {
	report, err := timereport.Build(context.Background(), nil, reportStart, reportEnd, noFetch)
	require.NoError(t, err)

	assert.Equal(t, 1, report.SchemaVersion)
	assert.Equal(t, reportStart, report.Period.Start)
	assert.Equal(t, reportEnd, report.Period.End)
	assert.Equal(t, "JST", report.Period.Timezone)
}
```

- [ ] **Step 2: テストを実行してコンパイルエラー（パッケージ未存在）を確認**

```bash
go test ./internal/timereport/...
```

Expected: `cannot find package` または `build failed` — パッケージがまだ存在しないため

- [ ] **Step 3: `internal/timereport/builder.go` を作成してテストを通す**

```go
// internal/timereport/builder.go
package timereport

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/hiraking/click-up-cli/internal/models"
)

// TaskFetcher はタスクIDからタスクメタデータを取得する関数型。
// client.ClickUpClient.GetTask をそのまま渡せるシグネチャ。
type TaskFetcher func(ctx context.Context, taskID string) (models.TaskSummary, error)

// Build は time entries を集計して TimeReport を返す。
// start は集計範囲の開始（含む）、end は終了（含まない）の半開区間。
func Build(
	ctx context.Context,
	entries []models.TimeEntry,
	start, end time.Time,
	fetch TaskFetcher,
) (models.TimeReport, error) {
	// 1. Dedup by entry ID
	seen := make(map[string]bool)
	var unique []models.TimeEntry
	for _, e := range entries {
		if !seen[e.ID] {
			seen[e.ID] = true
			unique = append(unique, e)
		}
	}

	// 2. Filter running timers (DurationMs < 0)
	var valid []models.TimeEntry
	for _, e := range unique {
		if e.DurationMs >= 0 {
			valid = append(valid, e)
		}
	}

	// 3. Clip each entry to [start, end) and discard zero-duration results
	type clippedEntry struct {
		entry  models.TimeEntry
		cStart time.Time
		cEnd   time.Time
		cDurMs int64
	}
	var clipped []clippedEntry
	for _, e := range valid {
		cs := maxTime(e.Start, start)
		ce := minTime(e.End, end)
		dur := ce.Sub(cs).Milliseconds()
		if dur <= 0 {
			continue
		}
		clipped = append(clipped, clippedEntry{e, cs, ce, dur})
	}

	// Task metadata in-memory cache
	taskCache := make(map[string]models.TaskSummary)

	// fetchCached fetches a task by ID, using the cache.
	fetchCached := func(taskID string) (models.TaskSummary, error) {
		if t, ok := taskCache[taskID]; ok {
			return t, nil
		}
		t, err := fetch(ctx, taskID)
		if err != nil {
			return models.TaskSummary{}, err
		}
		taskCache[taskID] = t
		return t, nil
	}

	// resolveTopLevel walks the parent chain to find the root task.
	var resolveTopLevel func(taskID string) (models.TaskSummary, error)
	resolveTopLevel = func(taskID string) (models.TaskSummary, error) {
		t, err := fetchCached(taskID)
		if err != nil {
			return models.TaskSummary{}, err
		}
		if t.ParentID == nil {
			return t, nil
		}
		return resolveTopLevel(*t.ParentID)
	}

	// Hierarchy accumulation maps (insertion-order tracking via slices)
	listOrder := []string{}
	listNames := make(map[string]string)
	listDur := make(map[string]int64)

	taskOrder := make(map[string][]string) // listID -> []topTaskIDs
	taskNames := make(map[string]string)
	taskDur := make(map[string]int64)
	taskSeen := make(map[string]bool)

	type bk struct{ top, rec string }
	bdOrder := make(map[string][]string) // topTaskID -> []recTaskIDs (in order)
	bdNames := make(map[string]string)   // recTaskID -> name
	bdDur := make(map[bk]int64)
	bdSeen := make(map[string]bool) // "topTaskID|recTaskID"

	var rows []models.TimeReportRow
	var totalDurMs int64

	for _, c := range clipped {
		e := c.entry
		totalDurMs += c.cDurMs

		// Resolve recorded task
		var recTask models.TaskSummary
		if e.TaskID == "" {
			recTask = models.TaskSummary{ID: e.ID, Name: e.TaskName, ListID: e.ListID, ListName: e.ListName}
		} else {
			t, err := fetchCached(e.TaskID)
			if err != nil {
				if errors.Is(err, client.ErrNotFound) {
					t = models.TaskSummary{ID: e.TaskID, Name: e.TaskName, ListID: e.ListID, ListName: e.ListName}
					taskCache[e.TaskID] = t
				} else {
					return models.TimeReport{}, fmt.Errorf("fetching task %s: %w", e.TaskID, err)
				}
			}
			recTask = t
		}

		// Resolve top-level task
		var topTask models.TaskSummary
		if e.TaskID == "" {
			topTask = recTask
		} else {
			var err error
			topTask, err = resolveTopLevel(e.TaskID)
			if err != nil {
				return models.TimeReport{}, fmt.Errorf("resolving top-level task for %s: %w", e.TaskID, err)
			}
		}

		// List resolution: top-level task > entry > "unknown"
		listID := topTask.ListID
		listName := topTask.ListName
		if listID == "" {
			listID = e.ListID
			listName = e.ListName
		}
		if listID == "" {
			listID = "unknown"
			listName = "unknown"
		}

		topTaskID := topTask.ID
		topTaskName := topTask.Name
		recTaskID := recTask.ID
		recTaskName := recTask.Name

		// Update list maps
		if _, ok := listNames[listID]; !ok {
			listOrder = append(listOrder, listID)
			listNames[listID] = listName
		}
		listDur[listID] += c.cDurMs

		// Update task maps
		if !taskSeen[topTaskID] {
			taskSeen[topTaskID] = true
			taskOrder[listID] = append(taskOrder[listID], topTaskID)
			taskNames[topTaskID] = topTaskName
		}
		taskDur[topTaskID] += c.cDurMs

		// Update breakdown maps
		bkStr := topTaskID + "|" + recTaskID
		if !bdSeen[bkStr] {
			bdSeen[bkStr] = true
			bdOrder[topTaskID] = append(bdOrder[topTaskID], recTaskID)
			bdNames[recTaskID] = recTaskName
		}
		bdDur[bk{topTaskID, recTaskID}] += c.cDurMs

		rows = append(rows, models.TimeReportRow{
			TimeEntryID:        e.ID,
			UserID:             e.UserID,
			UserName:           e.UserName,
			ListID:             listID,
			ListName:           listName,
			TopLevelTaskID:     topTaskID,
			TopLevelTaskName:   topTaskName,
			RecordedTaskID:     recTaskID,
			RecordedTaskName:   recTaskName,
			OriginalStart:      e.Start,
			OriginalEnd:        e.End,
			OriginalDurationMs: e.DurationMs,
			ClippedStart:       c.cStart,
			ClippedEnd:         c.cEnd,
			ClippedDurationMs:  c.cDurMs,
		})
	}

	// Build hierarchy slices from maps
	var lists []models.TimeReportList
	var topLevelCount, breakdownCount int

	for _, lid := range listOrder {
		var tasks []models.TimeReportTask
		for _, tid := range taskOrder[lid] {
			var breakdown []models.TimeReportBreakdown
			for _, rid := range bdOrder[tid] {
				breakdown = append(breakdown, models.TimeReportBreakdown{
					TaskID:      rid,
					TaskName:    bdNames[rid],
					DurationMin: bdDur[bk{tid, rid}] / 60000,
				})
				breakdownCount++
			}
			tasks = append(tasks, models.TimeReportTask{
				TaskID:      tid,
				TaskName:    taskNames[tid],
				DurationMin: taskDur[tid] / 60000,
				Breakdown:   breakdown,
			})
			topLevelCount++
		}
		lists = append(lists, models.TimeReportList{
			ListID:      lid,
			ListName:    listNames[lid],
			DurationMin: listDur[lid] / 60000,
			Tasks:       tasks,
		})
	}

	if lists == nil {
		lists = []models.TimeReportList{}
	}

	return models.TimeReport{
		SchemaVersion: 1,
		GeneratedAt:   time.Now(),
		Period: models.TimePeriod{
			Start:    start,
			End:      end,
			Timezone: start.Location().String(),
		},
		Summary: models.TimeReportSummary{
			TotalDurationMin:   totalDurMs / 60000,
			ListCount:          len(lists),
			TopLevelTaskCount:  topLevelCount,
			BreakdownTaskCount: breakdownCount,
		},
		Hierarchy: lists,
		Rows:      rows,
	}, nil
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
```

- [ ] **Step 4: テストを実行してすべて通ることを確認**

```bash
go test ./internal/timereport/... -v
```

Expected: すべてのテスト PASS

- [ ] **Step 5: 全テストが通ることを確認**

```bash
go test ./...
```

Expected: すべての既存テストも含めて PASS

- [ ] **Step 6: コミット**

```bash
git add internal/timereport/builder.go internal/timereport/builder_test.go
git commit -m "feat: add timereport builder with dedup, clip, parent chain resolution

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 5: time-report CLI コマンド

**Files:**
- Create: `cmd/clickup/time_report.go`
- Modify: `cmd/clickup/main.go`

- [ ] **Step 1: `cmd/clickup/time_report.go` を作成**

```go
// cmd/clickup/time_report.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/hiraking/click-up-cli/internal/dateparse"
	"github.com/hiraking/click-up-cli/internal/timereport"
)

func newTimeReportCmd() *cobra.Command {
	var flagStart, flagEnd, flagOutput string
	var flagRows bool

	cmd := &cobra.Command{
		Use:   "time-report",
		Short: "Aggregate time entries and output a JSON report",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			start, err := dateparse.ParseISO(flagStart, "start")
			if err != nil {
				return err
			}
			end, err := dateparse.ParseISO(flagEnd, "end")
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

			// --rows のデフォルト: --output あり → 含める、なし → 含めない
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

- [ ] **Step 2: `cmd/clickup/main.go` に time-report コマンドを登録**

`rootCmd.AddCommand(newUpdateTaskCmd())` の行の後に追加：

```go
	rootCmd.AddCommand(newTimeReportCmd())
```

完成形：

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
	rootCmd.AddCommand(newTimeReportCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: ビルドと全テストが通ることを確認**

```bash
go build ./... && go test ./...
```

Expected: ビルド成功・全テスト PASS

- [ ] **Step 4: コミット**

```bash
git add cmd/clickup/time_report.go cmd/clickup/main.go
git commit -m "feat: add time-report command

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 6: README 更新

**Files:**
- Modify: `README.md`

- [ ] **Step 1: README.md の「コマンドリファレンス」セクションに `time-report` を追加**

`### \`update-task\` — タスクを更新` セクションの後（`---` の直後）に以下を挿入：

```markdown
### `time-report` — 時間集計レポートを生成

```
clickup time-report --start <ISO8601> --end <ISO8601> [options]
```

指定期間の time entries を集計し、List → Top-level task → Breakdown のツリー構造で JSON レポートを出力する。

| オプション | 型 | 説明 |
|---|---|---|
| `--start <ISO8601>` | string | 集計開始日時（必須、半開区間の左端・含む） |
| `--end <ISO8601>` | string | 集計終了日時（必須、半開区間の右端・含まない） |
| `--output`, `-o <path>` | string | 出力ファイルパス。省略時は stdout |
| `--rows` | bool | normalized rows を含めるかどうか（後述のデフォルト参照） |

**`--rows` のデフォルト挙動:**

- `--output` あり かつ `--rows` 未指定 → rows **含める**
- `--output` なし かつ `--rows` 未指定 → rows **含めない**
- `--rows` / `--rows=false` 明示指定 → その値で上書き

**出力:** camelCase JSON。`hierarchy` フィールドに `List → Task → Breakdown` のツリー構造。

> **日付のタイムゾーンについて:** オフセットなしで渡した場合は **JST (+09:00)** として扱われる。

#### 使用例

```bash
# 週次レポートを stdout に出力（rows なし）
clickup time-report \
  --start "2026-04-27T00:00:00+09:00" \
  --end   "2026-05-04T00:00:00+09:00"

# ファイルに保存（rows も含める）
clickup time-report \
  --start "2026-04-27T00:00:00+09:00" \
  --end   "2026-05-04T00:00:00+09:00" \
  --output report.json

# rows を明示的に除外してファイル出力
clickup time-report \
  --start "2026-04-27T00:00:00+09:00" \
  --end   "2026-05-04T00:00:00+09:00" \
  --output report.json \
  --rows=false
```

---
```

- [ ] **Step 2: README.md の「エラーハンドリング」テーブルにレート制限の行を追加**

既存の API エラー行の後に追加：

```markdown
| レート制限（429） | `warning: rate limited, retrying in 60s (attempt 1/3)...` を stderr に出力してリトライ。3 回失敗で `HTTP Error (429): rate limit exceeded after 3 retries` |
```

- [ ] **Step 3: コミット**

```bash
git add README.md
git commit -m "docs: add time-report command to README

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```
