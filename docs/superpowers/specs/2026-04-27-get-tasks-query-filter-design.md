# get-tasks --query フィルタ 設計仕様

## 概要

`get-tasks` コマンドに `--query` フラグを追加し、タスク名と説明文をキーワードでクライアントサイドフィルタリングする。

## 背景

ClickUp API v2 にはテキスト検索パラメータが存在しない（`query`・`search`・`text` 系パラメータを持つエンドポイントなし）。そのため、API で全件取得した後にクライアントサイドで絞り込む方式を採用する。

## 変更ファイル

| ファイル | 変更内容 |
|---|---|
| `internal/client/client.go` | `GetTasksOptions` に `Query string` フィールド追加、`GetTasks` 内でflat listフィルタ適用 |
| `cmd/clickup/get_tasks.go` | `--query` フラグ追加、`GetTasksOptions.Query` に渡す |

## フィルタ仕様

- `--query` 未指定または空文字の場合 → 従来と同じ動作（全件返す）
- マッチ条件: `strings.ToLower(name + " " + description)` に `strings.ToLower(query)` が含まれる（case-insensitive部分一致）
- 検索対象フィールド: タスク名（`Name`）と説明文（`Description`）
- `Description` が `nil` の場合 → 空文字として扱う
- フィルタはツリー構築前の **flat list** に適用する

## ツリー構造の挙動

親タスクがマッチせず、サブタスクのみマッチした場合：

- 親タスク・マッチしないサブタスクは除外
- マッチしたサブタスクは `tree.Build` によりルートレベルに昇格して出力される

例：親タスク（マッチなし）+ サブタスク3件のうち1件がマッチ → そのサブタスクのみがルートタスクとして出力。

## 使用例

```bash
# "バグ" を名前または説明に含むタスクを取得
clickup get-tasks --query "バグ"

# リストとステータスフィルタと組み合わせ
clickup get-tasks --query "ログイン" --list "Backend" --status "in progress"
```

## 制約・注意事項

- 検索はAPI取得後にクライアントサイドで行うため、大量タスクがある場合でも全件フェッチされる（最大10ページ）
- 既存の `--list`・`--status`・`--due-after`・`--due-before` フィルタと組み合わせ可能
