# GitHub Copilot Instructions — ClickUpClient

## プロジェクト概要

ClickUp REST API v2 の Go 製薄いラッパー CLI ツール。  
エージェントや LLM から使うことを想定し、生 API レスポンスをそのまま渡すのではなく、  
必要最小限に整形した DTO を返すことを目的とする。

CLI コマンドの詳細（オプション・出力形式・使用例）は **README.md** を参照。

## ディレクトリ構成

```
cmd/clickup/              ← CLI エントリポイント (cobra)
internal/client/          ← HTTP クライアント・Raw デシリアライズ型・マッパー
internal/tree/            ← フラットリスト → ツリー構造変換
internal/timereport/      ← タイムエントリ集計レポート生成
internal/models/          ← エージェント向け整形済み DTO
internal/config/          ← 設定ローダー
internal/dateparse/       ← ISO 8601 パーサ
config.sample.json        ← 設定テンプレート（コミット対象）
~/.clickup/config.json    ← APIキー・設定（リポジトリ外）
```

## 設計方針

- **シンプルさ優先**: キャッシュ・複雑な抽象化は不要。薄いラッパーとして保つ
- **依存最小化**: `internal/` パッケージは標準ライブラリのみ使用。CLI 側は cobra / viper を使用
- **レイヤー分離**: Raw API 型は `internal/client/` 内に閉じ、外部には整形済み DTO のみ公開する
- **エラーハンドリング**: エラーは上位にそのまま伝播。過剰ラップしない
- **ページネーション**: 最大 10 ページ（1,000 件）を自動取得。上限到達時は stderr に警告を出す

## アーキテクチャ概要

```
CLI コマンド (cmd/clickup/)
  │  cobra でコマンド・フラグを定義し、入力を検証してクライアントを呼び出す
  ▼
ClickUpClient (internal/client/)
  │  HTTP 呼び出しのみ担当。Raw 型でデシリアライズし、mapper で DTO に変換して返す
  ▼
整形済み DTO (internal/models/)
  │  エージェント向けの camelCase JSON 出力用型。TaskSummary はツリーノードを兼ねる
  ▼
ユーティリティ
  ├─ internal/tree/       フラット配列 → 親子ツリーへの組み立て
  ├─ internal/timereport/ タイムエントリの集計・レポート生成
  ├─ internal/dateparse/  ISO 8601 文字列のパース（タイムゾーン対応）
  └─ internal/config/     設定ファイルの読み込みと解決
```

## 命名規則

- Raw 型: 未公開 (`rawTask` 等)。`internal/client/` 内にのみ存在
- 整形済み DTO: サフィックスなし（例: `TaskSummary`）
- JSON タグ: camelCase（例: `json:"dueDate"`）

## ClickUp API の注意点

- `date_created`, `due_date`, `start_date` 等は Unix ミリ秒の文字列（例: `"1567780450202"`）
- `time_estimate` は ms の整数文字列（例: `"8640000"`）
- `status.status` が表示名（例: `"in progress"`）、`priority.priority` が表示名（例: `"normal"`）
- `parent` が `null` = ルートタスク / 文字列 = 親タスクの ID
- GET /v2/team/{teamId}/task に `subtasks=true` を付けると全サブタスクもフラットに返る
- `due_date_gt` / `due_date_lt` フィルタは API 側で処理。親がフィルタ外でもサブタスクがマッチすると、そのサブタスクはルートとして返る

## 変更時のルール

- **README.md の更新**: CLI コマンド・オプション・出力形式・設定に変更を加えた場合は、`README.md` も合わせて更新する
- **copilot-instructions.md の更新**: アーキテクチャや設計方針を変更した場合は、本ファイルも合わせて更新する
- **Agent Skill 化の提案**: 複数ステップにわたる再利用可能な手順（セットアップ、検証、デバッグなど）が生まれたら、`.github/skills/` 配下への Agent Skill 化をユーザーに提案する
