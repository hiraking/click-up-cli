# time-report コマンド 設計仕様

## 概要

ClickUp time entries を指定期間で集計し、JSON レポートとして出力する `time-report` コマンドを追加する。

---

## CLI インターフェース

```
clickup time-report \
  --start "2026-04-27T00:00:00+09:00" \
  --end   "2026-05-04T00:00:00+09:00" \
  [--output report.json] \
  [--rows] [--rows=false]
```

### フラグ

| フラグ | 型 | 必須 | 説明 |
|---|---|---|---|
| `--start` | string | ✓ | 集計開始日時（ISO 8601、半開区間の左端・含む） |
| `--end` | string | ✓ | 集計終了日時（ISO 8601、半開区間の右端・含まない） |
| `--output`, `-o` | string | − | 出力ファイルパス。省略時は stdout |
| `--rows` | bool | − | rows を含めるかどうか（後述のデフォルト挙動を上書き） |

### `--rows` のデフォルト挙動

- `--output` あり かつ `--rows` 未指定 → rows **含める**
- `--output` なし かつ `--rows` 未指定 → rows **含めない**
- `--rows` 明示指定 → その値で上書き

---

## アーキテクチャ

### 新規ファイル

```
internal/models/time_report.go      TimeEntry / TimeReport / Row / Hierarchy DTO
internal/timereport/builder.go      集計ロジック（クリップ・親解決・階層構築）
internal/timereport/builder_test.go
internal/client/raw_time_entry.go   rawTimeEntry 型・toTimeEntry マッパー
cmd/clickup/time_report.go          time-report コマンド実装
```

### 変更ファイル

```
internal/client/client.go           GetTimeEntries() をインターフェースに追加
cmd/clickup/main.go                 time-report コマンド登録
README.md                           コマンドドキュメント追加
```

---

## データモデル

### `models.TimeEntry`（client レイヤーが返す処理済みエントリ）

```go
type TimeEntry struct {
    ID         string
    TaskID     string
    TaskName   string
    UserID     string
    UserName   string
    Billable   bool
    Start      time.Time
    End        time.Time
    DurationMs int64   // 元の duration（ms）
    // task_location から取得したリスト情報（フォールバック用）
    ListID     string
    ListName   string
}
```

### `models.TimeReport`（最終出力）

```go
type TimeReport struct {
    SchemaVersion int                `json:"schemaVersion"`
    GeneratedAt   time.Time          `json:"generatedAt"`
    Period        TimePeriod         `json:"period"`
    Summary       TimeReportSummary  `json:"summary"`
    Hierarchy     []TimeReportList   `json:"hierarchy"`
    Rows          []TimeReportRow    `json:"rows,omitempty"`  // nil の場合フィールドごと省略
}

type TimePeriod struct {
    Start    time.Time `json:"start"`
    End      time.Time `json:"end"`
    Timezone string    `json:"timezone"`  // start.Location().String()
}

type TimeReportSummary struct {
    TotalDurationMs    int64 `json:"totalDurationMs"`
    ListCount          int   `json:"listCount"`
    TopLevelTaskCount  int   `json:"topLevelTaskCount"`
    BreakdownTaskCount int   `json:"breakdownTaskCount"`
}

type TimeReportList struct {
    ListID     string           `json:"listId"`
    ListName   string           `json:"listName"`
    DurationMs int64            `json:"durationMs"`
    Tasks      []TimeReportTask `json:"tasks"`
}

type TimeReportTask struct {
    TaskID     string                  `json:"taskId"`
    TaskName   string                  `json:"taskName"`
    DurationMs int64                   `json:"durationMs"`
    Breakdown  []TimeReportBreakdown   `json:"breakdown"`
}

type TimeReportBreakdown struct {
    TaskID     string `json:"taskId"`
    TaskName   string `json:"taskName"`
    DurationMs int64  `json:"durationMs"`
}

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
    Billable           bool      `json:"billable"`
}
```

---

## 集計ロジック（`internal/timereport/builder.go`）

### Builder インターフェース

```go
// TaskFetcher はタスクメタデータ取得の抽象。client.GetTask を渡す。
type TaskFetcher func(ctx context.Context, taskID string) (models.TaskSummary, error)

func Build(
    ctx      context.Context,
    entries  []models.TimeEntry,
    start    time.Time,
    end      time.Time,
    fetch    TaskFetcher,
) (models.TimeReport, error)
```

### 処理フロー

1. **重複排除**: time entry ID をキーに dedup
2. **Running timer 除外**: `DurationMs < 0` のエントリを除外
3. **クリップ計算**:
   ```
   fetchBuffer = 3時間（固定）
   clippedStart = max(entry.Start, reportStart)
   clippedEnd   = min(entry.End, reportEnd)
   clippedDuration = max(0, clippedEnd - clippedStart)
   clippedDuration == 0 のエントリは除外
   ```
4. **Task Metadata 解決**（インメモリキャッシュ付き順次取得）:
   - エントリの TaskID から `fetch(ctx, taskID)` で `TaskSummary` 取得
   - `TaskSummary.ParentID != nil` の場合、再帰的に親を取得（キャッシュ済みなら再取得不要）
   - ルートに到達するまでたどり、top-level task を特定
5. **リスト解決**:
   ```
   primary:  topLevelTask.ListID / ListName
   fallback: entry.ListID / ListName
   fallback: "unknown"
   ```
6. **階層構築**:
   - List → TopLevelTask → BreakdownTask の Map を構築してから []スライスへ変換
   - durationMs は clippedDuration の合計
7. **rows 構築**: 各エントリを `TimeReportRow` に変換
8. **summary 集計**: totalDurationMs / 各カウント

### Task が存在しないエントリの扱い

タスクIDが空、または API がエラーを返した場合：
- ListID/ListName = "unknown"、TopLevelTask = 元の TaskName or entry ID でフォールバック（エラーは無視せず、上位に伝播）
- ただし 404 の場合は "unknown" にフォールバックしてエラーにしない

---

## HTTP クライアント拡張（`internal/client/`）

### `GetTimeEntries` オプション

```go
type GetTimeEntriesOptions struct {
    Start time.Time
    End   time.Time
}
```

### API 呼び出し

```
GET /v2/team/{teamId}/time_entries
  ?start_date={fetchStart_unix_ms}
  &end_date={fetchEnd_unix_ms}
  &include_location_names=true
```

`fetchStart = start - 3h`, `fetchEnd = end + 3h`

### raw 型（`raw_time_entry.go`）

```go
type rawTimeEntry struct {
    ID           string              `json:"id"`
    Task         *rawEntryTask       `json:"task"`
    User         rawEntryUser        `json:"user"`
    Billable     bool                `json:"billable"`
    Start        string              `json:"start"`  // Unix ms 文字列
    End          string              `json:"end"`    // Unix ms 文字列（running timer は "0" か空）
    Duration     string              `json:"duration"` // 負値 = running timer
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
    ListID   json.Number `json:"list_id"`   // API が int or string を返す
    ListName string      `json:"list_name"`
}
```

---

## 出力 JSON 例

```json
{
  "schemaVersion": 1,
  "generatedAt": "2026-05-03T14:00:00+09:00",
  "period": {
    "start": "2026-04-27T00:00:00+09:00",
    "end": "2026-05-04T00:00:00+09:00",
    "timezone": "Asia/Tokyo"
  },
  "summary": {
    "totalDurationMs": 126000000,
    "listCount": 3,
    "topLevelTaskCount": 12,
    "breakdownTaskCount": 28
  },
  "hierarchy": [
    {
      "listId": "list_1",
      "listName": "Product Development",
      "durationMs": 72000000,
      "tasks": [
        {
          "taskId": "task_parent_1",
          "taskName": "新料金ページ改善",
          "durationMs": 28800000,
          "breakdown": [
            { "taskId": "subtask_1", "taskName": "UI実装", "durationMs": 18000000 }
          ]
        }
      ]
    }
  ]
}
```

`--rows` 有効時は `rows` フィールドが追加される。無効時はフィールドごと省略（`omitempty`）。

---

## 初期仕様の対象外

- タスクメタデータのディスクキャッシュ
- 並列 API 取得（rate limit 対応）
- Running timer の速報レポート
- 週次・月次などのショートカットオプション
- CSV / HTML などの非 JSON 出力
