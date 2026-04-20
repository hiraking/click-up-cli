# GitHub Copilot Instructions — ClickUpClient

## プロジェクト概要

ClickUp REST API v2 の C# 製薄いラッパーライブラリ + CLI ツール。  
エージェントや LLM から使うことを想定し、生 API レスポンスをそのまま渡すのではなく、  
必要最小限に整形した DTO を返すことを目的とする。

## ディレクトリ構成

```
src/ClickUpClient/        ← メインライブラリ (net10.0 classlib)
  Http/                   ← API 呼び出し層 (ClickUpHttpClient)
  Raw/                    ← API 生レスポンス用デシリアライズモデル
  Models/                 ← エージェント向け整形済み DTO
  Mapping/                ← Raw → Models 変換ロジック
  Tree/                   ← タスクリスト → ツリー構造変換
src/ClickUpCli/           ← CLI ツール (net10.0 Console App)
  Program.cs              ← エントリポイント、コマンド定義 (System.CommandLine)
  Config.cs               ← AppConfig モデル + ConfigLoader
  config.json             ← APIキー・設定 (gitignore対象)
  config.sample.json      ← 設定テンプレート (コミット対象)
tests/ClickUpClient.Tests/ ← xUnit テスト
```

## 設計方針

- **シンプルさ優先**: キャッシュ・設定管理・複雑な抽象化は不要。薄いラッパーとして保つ
- **依存最小化**: ライブラリ側は外部 NuGet パッケージを追加しない。`System.Text.Json` のみ使用。CLI 側は `System.CommandLine` を使用
- **DI フレンドリー**: `ClickUpHttpClient` は `HttpClient` をコンストラクタで受け取る
- **エラーハンドリング**: `HttpRequestException` をそのまま上位に伝播。過剰ラップしない
- **ページネーション**: `page` パラメータの口は用意するが、デフォルトは page=0 のみ取得

## 命名規則

- Raw モデル: `Raw` プレフィックス（例: `RawTask`, `RawTaskStatus`）
- 整形済み DTO: サフィックスなし（例: `TaskSummary`）
- 変換クラス: `Mapper` サフィックス（例: `TaskMapper`）
- ツリー構築: `Builder` サフィックス（例: `TaskTreeBuilder`）
- JSON プロパティ: `[JsonPropertyName("snake_case")]` で明示

## 主要クラスの責務

### `IClickUpClient` / `ClickUpHttpClient`

ClickUp API への HTTP 呼び出しのみ担当。  
`Authorization: {apiKey}` ヘッダーを付与し、`System.Text.Json` でデシリアライズして返す。  
整形・変換はしない。

### `Raw/` 配下のモデル

API レスポンスの JSON をそのまま受け取るためのモデル。  
全フィールドを再現する必要はなく、`TaskSummary` 変換に必要なフィールドのみ定義する。

### `TaskSummary` (Models/)

エージェント向けの整形済み DTO。`TaskSummary` 自体がツリーノードを兼ねる。  
`Subtasks: IReadOnlyList<TaskSummary>` を持ち、子タスクをネストで保持する。  
Unix ms 文字列は `DateTimeOffset` に変換済み。Status / Priority は表示名の文字列として保持。  
個人利用のため `Assignees` は含まない。

### `TaskMapper` (Mapping/)

`RawTask` → `TaskSummary` への変換。Subtasks は空リストで生成する（ツリー構築前）。  
static クラスまたは static メソッドとして実装。

### `TaskTreeBuilder` (Tree/)

`IEnumerable<RawTask>` を受け取り、`IReadOnlyList<TaskSummary>` としてツリー構造を返す。  
`parent == null` のタスクをルートとして扱い、再帰的に多段ネストを構築する。

## ClickUp API の注意点

- `date_created`, `due_date` 等は Unix ミリ秒の文字列（例: `"1567780450202"`）
- `time_estimate` は ms の文字列（例: `"8640000"`）
- `status.status` が表示名（例: `"in progress"`）、`priority.priority` が表示名（例: `"normal"`）
- `parent` が `null` = ルートタスク / 文字列 = 親タスクの ID
- GET /v2/team/{teamId}/task に `subtasks=true` を付けると全サブタスクもフラットに返る
- `due_date_gt` / `due_date_lt` フィルタは API 側で処理。親がフィルタ外でもサブタスクがマッチすると、そのサブタスクはルートとして返る

## ClickUpCli の責務

- `AppConfig`: `config.json` のデシリアライズモデル（apiKey / teamId / lists マッピング）
- `ConfigLoader`: `AppContext.BaseDirectory/config.json` を読み込む。必須フィールド検証あり
- `Program.cs`: `System.CommandLine` で `get-tasks` / `get-task` コマンドを定義。出力は camelCase JSON（日本語はUnicodeエスケープしない）
- `config.json` は gitignore 対象。`config.sample.json` がテンプレート

## 変更時のルール

- **README.md の更新**: CLI コマンド・オプション・出力形式・設定に変更を加えた場合は、`README.md` も合わせて更新する
- **Agent Skill 化の提案**: 複数ステップにわたる再利用可能な手順（セットアップ、検証、デバッグなど）が生まれたら、`.github/skills/` 配下への Agent Skill 化をユーザーに提案する

## タイムトラッキング

現時点では対象外。後で `IClickUpClient` にメソッドを追加することで対応可能な設計にしている。
