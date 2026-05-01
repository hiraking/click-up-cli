# Design: create-task に --task-type オプションを追加

Date: 2026-05-01

## 概要

`create-task` コマンドに `--task-type` オプションを追加する。ユーザーは `milestone` / `project` / `book` のいずれかの文字列を渡し、CLI 内部でそれを ClickUp API の `custom_item_id`（数値）に変換してリクエストする。

## 背景

ClickUp API v2 の Create Task エンドポイントはリクエストボディに `custom_item_id` フィールドを受け付ける。このチームでは以下の固定マッピングを使用する：

| 文字列       | custom_item_id | 説明           |
|------------|---------------|---------------|
| milestone  | 1             | マイルストーン |
| project    | 1001          | プロジェクト   |
| book       | 1003          | 書籍           |

外部（CLI ユーザー・AI エージェント）からは数値 ID ではなく意味のある文字列で指定できるようにする。

## 設計

### アーキテクチャ

既存の `--priority` オプションと完全に同じ層構造で実装する：

```
cmd (文字列 → TaskType 変換) → models.CreateTaskRequest → mapper → rawCreateTaskBody (int) → API
```

### 変更ファイル一覧

#### 1. `internal/models/create_task.go`

`TaskPriority` と同じ要領で `TaskType` 型と定数を追加：

```go
type TaskType int

const (
    TaskTypeMilestone TaskType = 1
    TaskTypeProject   TaskType = 1001
    TaskTypeBook      TaskType = 1003
)
```

`CreateTaskRequest` に `CustomItemID *TaskType` フィールドを追加。

#### 2. `internal/client/raw_create.go`

`rawCreateTaskBody` に追加：

```go
CustomItemID *int `json:"custom_item_id,omitempty"`
```

#### 3. `internal/client/mapper.go`

`mapToRawCreateBody` に変換ロジックを追加：

```go
if req.CustomItemID != nil {
    id := int(*req.CustomItemID)
    body.CustomItemID = &id
}
```

#### 4. `cmd/clickup/create_task.go`

`parsePriority` と同じ形式で `parseTaskType` を追加し、`--task-type` フラグを登録：

```go
func parseTaskType(s string) (models.TaskType, error) {
    switch strings.ToLower(s) {
    case "milestone": return models.TaskTypeMilestone, nil
    case "project":   return models.TaskTypeProject, nil
    case "book":      return models.TaskTypeBook, nil
    default:
        return 0, fmt.Errorf("Error: Invalid task type '%s'. Use milestone, project, or book.", s)
    }
}
```

#### 5. `README.md`

`create-task` オプション表に `--task-type <name>` を追記。

### エラーハンドリング

不正な文字列が渡された場合：

```
Error: Invalid task type 'foo'. Use milestone, project, or book.
```

### テスト

- `parseTaskType` の単体テスト（有効値3種 + 無効値）を `cmd/clickup/` または `internal/models/` に追加
- `mapToRawCreateBody` のテストに `CustomItemID` ケースを追加

## 対象外

- `update-task` コマンドへの `--task-type` 追加（別タスクとして扱う）
- タスクタイプ一覧の動的取得（Get Custom Task Types API の利用）
