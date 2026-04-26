# ClickUp CLI — Go 移行設計書

**日付:** 2026-04-26  
**対象:** C# (.NET 10) → Go (cobra + viper) への完全書き直し

---

## 概要

C# で実装された ClickUp CLI（ライブラリ + CLI ツール）を Go に移行する。  
既存の機能・コマンドインターフェース・出力形式を完全に維持しつつ、C# コードは削除する。  
cobra でコマンド定義、viper で設定ファイル読み込みを行う。

---

## プロジェクト構造

```
github.com/hiraking/click-up-client   ← Go module 名

cmd/clickup/
  main.go            ← エントリポイント、cobra root command 初期化
  get_tasks.go       ← get-tasks サブコマンド定義
  get_task.go        ← get-task サブコマンド定義
  create_task.go     ← create-task サブコマンド定義

internal/
  client/
    client.go        ← ClickUpClient interface + HTTP 実装
    raw_task.go      ← RawTask, RawTaskStatus, RawPriority, RawListRef, RawAssignee
    raw_response.go  ← RawGetTasksResponse
    raw_create.go    ← RawCreateTaskBody
  models/
    task_summary.go  ← TaskSummary struct（エージェント向け整形済み DTO）
    create_task.go   ← CreateTaskRequest struct, TaskPriority 定数
  mapping/
    task_mapper.go   ← RawTask → TaskSummary 変換
  tree/
    builder.go       ← フラットリスト → ツリー構造変換
  config/
    config.go        ← AppConfig struct + Load() 関数
  dateparse/
    parse.go         ← ISO 8601 文字列パース（オフセットなし → JST +09:00）

tests/
  mapping/           ← task_mapper_test.go
  tree/              ← builder_test.go
  dateparse/         ← parse_test.go

config.sample.json   ← 設定テンプレート（コミット対象）
README.md
go.mod
go.sum
```

削除対象: `src/`, `tests/` (C#), `ClickUpClient.slnx`, `*.csproj`

---

## 依存パッケージ

| パッケージ | 用途 |
|---|---|
| `github.com/spf13/cobra` | CLI コマンド定義 |
| `github.com/spf13/viper` | 設定ファイル読み込み |
| `github.com/stretchr/testify` | テストアサーション (assert/require) |

HTTP クライアント・JSON シリアライズは標準ライブラリ (`net/http`, `encoding/json`) のみ使用。

---

## アーキテクチャ詳細

### コマンド構造

```
clickup
├── get-tasks  [--list <name>...] [--status <name>...] [--due-after <ISO8601>] [--due-before <ISO8601>] [--no-subtasks]
├── get-task   <taskId>
└── create-task <name> --list <name> [--description <text>] [--parent <id>]
                       [--status <name>] [--priority urgent|high|normal|low]
                       [--due-date <ISO8601>] [--start-date <ISO8601>] [--time-estimate <分>]
```

各サブコマンドは `cmd/clickup/` 内の別ファイルに `newGetTasksCmd()` などの関数として定義し、`main.go` で `rootCmd.AddCommand(...)` する。

### 設定ファイル読み込み（viper）

- `config.json` をバイナリ実行ディレクトリ（`filepath.Dir(os.Executable())`）から固定パスで読む
- `viper.SetConfigFile(path)` + `viper.ReadInConfig()` を使用
- 必須フィールド (`apiKey`, `teamId`) が空の場合はエラーメッセージを stderr に出力して exit 1

```json
{
  "apiKey": "pk_YOUR_API_KEY_HERE",
  "teamId": "YOUR_TEAM_ID_HERE",
  "lists": {
    "work": "LIST_ID_1",
    "study": "LIST_ID_2"
  }
}
```

### HTTP クライアント

- `internal/client/client.go` に `ClickUpClient` interface と `httpClient` struct を定義
- `Authorization: {apiKey}` ヘッダーを全リクエストに付与
- Base URL: `https://api.clickup.com/api/`
- デシリアライズ: `encoding/json` + `json:"snake_case"` タグ
- エラー時は `fmt.Errorf("HTTP %d: %s", statusCode, body)` 形式のエラーを返す

### JSON 出力

- `encoding/json` の `json:"camelCase"` タグで出力フィールド名を制御
- `json.MarshalIndent` でインデントあり（C# の `WriteIndented: true` に相当）
- 日本語 Unicode エスケープなし: `json.NewEncoder(os.Stdout)` + `enc.SetEscapeHTML(false)` を使用
- 日付は `time.Time` を RFC 3339 形式で出力（`time.RFC3339Nano`）

### TaskSummary（Go版）

```go
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

### 日付パース（dateparse）

- 複数の ISO 8601 フォーマットを順に試行して `time.Time` を返す
- オフセット（`Z` または `+HH:MM` / `-HH:MM`）が含まれない場合は JST (+09:00) として扱う
- オフセット検出: `Z` で終わる、または 10文字目以降に `+` / `-` が存在するか

### ツリー構築（tree/builder.go）

1. フラットな `[]RawTask` を全て `TaskSummary`（Subtasks 空）に変換し ID → TaskSummary の map を作成
2. 各タスクの `parent` フィールドを見て children map (`parentID → []TaskSummary`) を構築
3. `parent == nil` または親が map に存在しないタスクをルートとして再帰的に Subtasks を埋める
4. ルートタスクを ID 順にソートして返す

---

## エラーハンドリング

- 全サブコマンドで `cobra.Command.SilenceUsage = true` を設定（エラー時に usage が表示されないよう）
- エラーメッセージは stderr に出力して exit 1
- エラー形式は現行 C# 版と同一:

| ケース | メッセージ |
|---|---|
| `config.json` が見つからない | `Error: config.json not found at '...'` |
| 不明なリスト名 | `Error: Unknown list name 'foo'. Available: work, study` |
| 日付フォーマット不正 | `Error: '--due-after' value '...' is not a valid ISO 8601 datetime.` |
| 不正な優先度 | `Error: Invalid priority 'foo'. Use urgent, high, normal, or low.` |
| API エラー | `HTTP Error (404): ...` |

---

## テスト方針

- `internal/mapping/`: `RawTask` → `TaskSummary` 変換の正常系・null フィールド
- `internal/tree/`: フラットリスト→ツリー変換、ルート判定、多段ネスト
- `internal/dateparse/`: オフセットあり/なし/Z/+09:00/不正値のパース
- `cmd/clickup/`: 不明リスト名・不正優先度のバリデーションエラー（設定ファイルをモックして実行）

---

## ビルド・セットアップ

```bash
# ビルド
go build -o out/clickup ./cmd/clickup

# テスト
go test ./...
```

設定ファイルは `config.json` をバイナリと同じディレクトリに配置する（`config.sample.json` を参照）。

---

## 移行スコープ外

- ページネーション（page=0 のみ、現行踏襲）
- タイムトラッキング
- C# テストの Go 移植（同等テストを新規作成）
