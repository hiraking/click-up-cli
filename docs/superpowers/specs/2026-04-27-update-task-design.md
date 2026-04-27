# update-task コマンド 設計仕様

**日付:** 2026-04-27  
**ステータス:** 承認済み

---

## 概要

ClickUp タスクを CLI から更新する `update-task` コマンドを追加する。既存の `create-task` / `get-task` コマンドのパターンに倣い、更新後のタスクを JSON で出力する。

---

## コマンドインターフェース

```
update-task <taskId> [flags]
```

### フラグ

| フラグ | 型 | 説明 |
|---|---|---|
| `--name` | string | タスク名 |
| `--description` | string | 説明 |
| `--status` | string | ステータス名 |
| `--priority` | string | `urgent` / `high` / `normal` / `low` |
| `--due-date` | string | ISO8601（例: `2026-05-01T18:00+09:00`）|
| `--start-date` | string | ISO8601 |
| `--time-estimate` | int | 見積もり時間（分）|
| `--parent` | string | 親タスク ID |
| `--clear` | []string | クリアするフィールド名（繰り返し可）|

### `--clear` フラグの仕様

`--clear FIELD` を指定すると、対象フィールドを API 上でクリア（null / 削除）する。複数フィールドを同時にクリアする場合は繰り返し指定する。

**受け付けるフィールド名:**  
`description`, `status`, `priority`, `due-date`, `start-date`, `time-estimate`, `parent`

> `name` はクリア不可。ClickUp API でタスク名は必須フィールドのため。

**使用例:**
```bash
# 見積もり時間をクリア
update-task abc123 --clear time-estimate

# 期日と説明を同時にクリア
update-task abc123 --clear due-date --clear description

# 名前を変更しつつ期日をクリア
update-task abc123 --name "新しい名前" --clear due-date
```

不正なフィールド名を指定した場合はエラーを返す:
```
Error: invalid field for --clear: "foo". Accepted: description, status, priority, due-date, start-date, time-estimate, parent
```

フラグを何も指定しない場合もエラーを返す:
```
Error: no fields specified to update.
```

---

## アーキテクチャ

### 新規ファイル

- `cmd/clickup/update_task.go` — コマンド定義・フラグ処理・入力検証

### 変更ファイル

| ファイル | 変更内容 |
|---|---|
| `internal/models/create_task.go` | `UpdateTaskRequest` 型を追加 |
| `internal/client/raw_types.go` | `rawUpdateTaskBody` 型を追加 |
| `internal/client/client.go` | `UpdateTask()` メソッドを追加（インターフェース含む）|
| `cmd/clickup/main.go` | `update-task` コマンドを登録 |

---

## データフロー

```
update-task abc123 --name "foo" --clear due-date
         ↓
1. cobra: Changed() でフラグ指定を検出
2. --clear の値を検証（有効フィールド名かチェック）
3. UpdateTaskRequest を構築（変更フィールドのみポインタ設定、クリア対象はフラグで別管理）
4. client.UpdateTask(ctx, taskID, req) 呼び出し
5. PUT /v2/task/{taskId} にリクエスト
6. レスポンスを TaskSummary にマッピング（既存 mapper 流用）
7. JSON 出力（既存 printJSON ヘルパー流用）
```

---

## モデル定義

### `models.UpdateTaskRequest`

```go
type UpdateTaskRequest struct {
    Name         *string
    Description  *string
    Status       *string
    Priority     *TaskPriority
    DueDate      *time.Time
    StartDate    *time.Time
    TimeEstimate *time.Duration
    Parent       *string
    ClearFields  []string  // クリア対象フィールド名のリスト
}
```

フラグが指定されたフィールドのみポインタに値を入れる（`Changed()` パターン）。`ClearFields` は `--clear` で指定されたフィールド名を格納する。

### `rawUpdateTaskBody`

ClickUp API の PUT `/v2/task/{taskId}` ボディに対応する型。`rawCreateTaskBody` と構造が類似するため、同ファイルに定義する。

**注意:** 通常フィールドは `omitempty` で nil を除外するが、クリア対象フィールドは明示的に null / 0 を API に送る必要がある。`ClearFields` に含まれるフィールドは `omitempty` を使わない専用フィールドで上書きするか、カスタムマーシャラーで対応する。実装時に ClickUp API の各フィールドのクリア方法（null / 0 / 空文字列）を `@references/clickup-api-v2-reference.json` で確認すること。

---

## エラーハンドリング

| ケース | 挙動 |
|---|---|
| `--clear` に不正なフィールド名 | `Error: invalid field for --clear: "foo". Accepted: ...` |
| `--priority` に不正な値 | `Error: Invalid priority 'xxx'. Use urgent, high, normal, or low.` |
| 日付フォーマット不正 | `Error: invalid due-date: ...`（既存 dateparse パターン）|
| フラグ未指定 | `Error: no fields specified to update.` |
| API エラー（404 等） | HTTP ステータスとボディを含めたエラー（既存パターン）|

---

## テスト方針

- `internal/client/mapper_test.go` に `TestMapToRawUpdateBody_*` を追加
  - フィールド変更のマッピング検証
  - `ClearFields` が正しくボディに反映されることの検証
  - 日付・duration のミリ秒変換検証
- `cmd/clickup/update_task.go` の `--clear` フィールド検証ロジックのユニットテスト
- `internal/client/client_test.go` に `TestUpdateTask_*` を追加（API 呼び出しのモックテスト）

---

## ドキュメント・ヘルプ

- cobra の `Long` フィールドに `--clear` の動作仕様を明記する
- `README.md` の `update-task` セクションに `--clear` の使用例と仕様を記載する
