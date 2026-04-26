# GitHub Copilot Instructions — ClickUpClient

## プロジェクト概要

ClickUp REST API v2 の Go 製薄いラッパー CLI ツール。  
エージェントや LLM から使うことを想定し、生 API レスポンスをそのまま渡すのではなく、  
必要最小限に整形した DTO を返すことを目的とする。

## ディレクトリ構成

```
cmd/clickup/              ← CLI エントリポイント (cobra)
  main.go                 ← rootCmd 定義、SilenceUsage/SilenceErrors
  helpers.go              ← loadConfig(), printJSON(), availableListNames()
  get_task.go             ← get-task コマンド
  get_tasks.go            ← get-tasks コマンド
  create_task.go          ← create-task コマンド、parsePriority()
internal/client/          ← HTTP クライアント + 未公開 Raw 型 + マッパー
  client.go               ← ClickUpClient インターフェース、httpClient 実装
  raw_types.go            ← 未公開 raw デシリアライズ型（rawTask 等）
  raw_create.go           ← 未公開 rawCreateTaskBody
  mapper.go               ← toSummary(), mapToRawCreateBody()
  mapper_test.go
internal/tree/            ← フラットリスト → ツリー構造変換
  builder.go              ← Build([]TaskSummary) []TaskSummary
  builder_test.go
internal/models/          ← エージェント向け整形済み DTO
  task_summary.go         ← TaskSummary（Subtasks []TaskSummary を含む）
  create_task.go          ← CreateTaskRequest、TaskPriority 定数
internal/config/          ← 設定ローダー
  config.go               ← AppConfig、Load(path string)
internal/dateparse/       ← ISO 8601 パーサ
  parse.go                ← ParseISO(value, optionName string)
  parse_test.go
config.sample.json        ← 設定テンプレート（コミット対象）
~/.clickup/config.json    ← APIキー・設定（リポジトリ外）
```

## 設計方針

- **シンプルさ優先**: キャッシュ・設定管理・複雑な抽象化は不要。薄いラッパーとして保つ
- **依存最小化**: `internal/` パッケージは標準ライブラリのみ使用。CLI 側は cobra / viper を使用
- **未公開 Raw 型**: `internal/client/` 内にのみ存在し、外部パッケージには公開しない
- **エラーハンドリング**: エラーは上位にそのまま伝播。過剰ラップしない
- **ページネーション**: page=0 のみ取得（`GetTasksOptions.Page` フィールドは存在するが常に 0）

## 命名規則

- Raw 型: 未公開 (`rawTask`, `rawTaskStatus` 等)。`internal/client/` 内にのみ存在
- 整形済み DTO: サフィックスなし（例: `TaskSummary`）
- 変換関数: `internal/client/mapper.go` に未公開関数として実装
- ツリー構築: `internal/tree/builder.go` の `Build()` 関数
- JSON タグ: camelCase（例: `json:"dueDate"`）

## 主要コンポーネントの責務

### `ClickUpClient` インターフェース / `httpClient` 実装 (`internal/client/client.go`)

ClickUp API への HTTP 呼び出しのみ担当。  
`Authorization: {apiKey}` ヘッダーを付与し、`encoding/json` でデシリアライズして返す。  
30 秒タイムアウト設定済み。整形・変換はしない。

### Raw 型 (`internal/client/raw_types.go`, `raw_create.go`)

API レスポンスの JSON をそのまま受け取るための未公開型。  
`TaskSummary` 変換に必要なフィールドのみ定義する。

### `TaskSummary` (`internal/models/task_summary.go`)

エージェント向けの整形済み DTO。`TaskSummary` 自体がツリーノードを兼ねる。  
`Subtasks []TaskSummary` を持ち、子タスクをネストで保持する（`omitempty` なし — 空配列として出力）。  
Unix ms 文字列は `time.Time` に変換済み。Status / Priority は表示名の文字列として保持。  
`StartDate` / `DueDate` はともに `*time.Time`（`omitempty` あり）。  
個人利用のため `Assignees` は含まない。

### `toSummary` / `mapToRawCreateBody` (`internal/client/mapper.go`)

`rawTask` → `TaskSummary`、`CreateTaskRequest` → `rawCreateTaskBody` への変換。  
`hasTimeComponent` は UTC に正規化してから判定する。

### `Build` (`internal/tree/builder.go`)

`[]TaskSummary` を受け取り、ルートタスクをエントリとするツリー構造を返す。  
`parent == nil` のタスクをルートとして扱い、再帰的に多段ネストを構築する。  
page=0 のみ取得のため、親が結果セット外のタスクはルートに昇格する（決定的な順序）。

### `ParseISO` (`internal/dateparse/parse.go`)

`ParseISO(value, optionName string) (time.Time, error)` — 2 引数シグネチャ必須。  
オフセットなし文字列は JST (`Asia/Tokyo`) にフォールバックする。  
エラーメッセージに `optionName` を含める（例: `"'--due-after' value '...' is not a valid ISO 8601 datetime"`）。

### `AppConfig` / `Load` (`internal/config/config.go`)

viper で `~/.clickup/config.json` を読み込む。`APIKey` / `TeamID` フィールド（mapstructure タグ: `apiKey` / `teamId`）。  
ファイル未存在検出は `errors.As(err, &pathErr)` (`*os.PathError`) を使用する。

## ClickUp API の注意点

- `date_created`, `due_date`, `start_date` 等は Unix ミリ秒の文字列（例: `"1567780450202"`）
- `time_estimate` は ms の整数文字列（例: `"8640000"`）
- `status.status` が表示名（例: `"in progress"`）、`priority.priority` が表示名（例: `"normal"`）
- `parent` が `null` = ルートタスク / 文字列 = 親タスクの ID
- GET /v2/team/{teamId}/task に `subtasks=true` を付けると全サブタスクもフラットに返る
- `due_date_gt` / `due_date_lt` フィルタは API 側で処理。親がフィルタ外でもサブタスクがマッチすると、そのサブタスクはルートとして返る

## CLI コマンド

| コマンド | 主なオプション |
|---|---|
| `get-tasks` | `--list`（複数可）, `--status`, `--due-after`, `--due-before`, `--no-subtasks` |
| `get-task <taskId>` | なし |
| `create-task` | `--list`（必須）, `--description`, `--parent`, `--status`, `--priority`, `--due-date`, `--start-date`, `--time-estimate` |

出力: camelCase JSON、日本語は Unicode エスケープしない（`SetEscapeHTML(false)`）。

## 変更時のルール

- **README.md の更新**: CLI コマンド・オプション・出力形式・設定に変更を加えた場合は、`README.md` も合わせて更新する
- **copilot-instructions.md の更新**: アーキテクチャや設計方針を変更した場合は、本ファイルも合わせて更新する
- **Agent Skill 化の提案**: 複数ステップにわたる再利用可能な手順（セットアップ、検証、デバッグなど）が生まれたら、`.github/skills/` 配下への Agent Skill 化をユーザーに提案する

## タイムトラッキング

現時点では対象外。後で `ClickUpClient` インターフェースにメソッドを追加することで対応可能な設計にしている。
