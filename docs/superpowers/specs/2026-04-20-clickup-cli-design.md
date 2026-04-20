# ClickUp CLI — Design Spec

**Date:** 2026-04-20  
**Status:** Approved

---

## Problem

`IClickUpClient` ライブラリはC#コードとして利用することを前提としているが、  
AIエージェントや外部スクリプトから直接呼び出すことができない。  
ClickUp APIをAIエージェントから叩けるように、薄いCLIラッパーを追加する。

---

## Approach

`System.CommandLine`（Microsoft公式NuGet）を使い、`IClickUpClient` の2メソッドを  
CLIサブコマンドとして公開する。出力はすべてJSON（stdoutへ）。  
設定ファイル（APIキー・teamId・listマッピング）はリポジトリ内に置くが `.gitignore` で除外する。

---

## Project Structure

```
src/ClickUpCli/
  ClickUpCli.csproj     ← net10.0 Console App
  Program.cs            ← エントリポイント、System.CommandLine コマンド定義
  Config.cs             ← AppConfig モデル + 設定ファイル読み込み
  config.json           ← 実際の設定（gitignore 対象）
  config.sample.json    ← サンプル設定（コミット対象）
```

---

## Commands

### `get-tasks`

```
clickup get-tasks [--list <name>...] [--status <name>...] [--due-after <ISO8601>] [--due-before <ISO8601>] [--no-subtasks]
```

| オプション | 型 | 説明 |
|---|---|---|
| `--list` | string[] | `config.json` の `lists` マッピングで解決されるリスト名。複数指定可。省略時は全リスト |
| `--status` | string[] | フィルタするステータス名（例: `"in progress"`）。複数指定可 |
| `--due-after` | ISO8601 string | この日時より後の due_date を持つタスクに絞り込む |
| `--due-before` | ISO8601 string | この日時より前の due_date を持つタスクに絞り込む |
| `--no-subtasks` | flag | 指定するとサブタスクを取得しない（デフォルト: サブタスクあり） |

`IClickUpClient.GetTasksAsync` を呼び出し、結果のツリー（`IReadOnlyList<TaskSummary>`）をJSONで返す。

### `get-task`

```
clickup get-task <taskId>
```

| 引数 | 型 | 説明 |
|---|---|---|
| `taskId` | string | ClickUp タスクID（必須） |

`IClickUpClient.GetTaskAsync` を呼び出し、単一の `TaskSummary` をJSONで返す。

---

## Configuration

### `config.json`（gitignore 対象）

```json
{
  "apiKey": "pk_xxxxxxxx",
  "teamId": "12345678",
  "lists": {
    "dev": "abc123456",
    "ops": "def789012"
  }
}
```

### `config.sample.json`（コミット対象）

値をダミーにしたサンプル。開発者がコピーして使う。

### 読み込みロジック

- 実行ファイルのディレクトリ（`AppContext.BaseDirectory`）の `config.json` を読む
- ファイルが存在しない場合は stderr にエラーを出して exit code 1 で終了

---

## Output Format

すべての正常出力は **stdout に JSON** として書き出す。

- `get-tasks`: `TaskSummary[]` のJSON配列
- `get-task`: `TaskSummary` のJSONオブジェクト

シリアライズは `System.Text.Json` を使用し、プロパティ名は **camelCase**（`JsonNamingPolicy.CamelCase`）。

---

## Error Handling

| ケース | 動作 |
|---|---|
| `config.json` が見つからない | stderr にメッセージ、exit code 1 |
| 不明なリスト名（`--list` で指定） | stderr にメッセージ、exit code 1 |
| `HttpRequestException` | stderr にメッセージ（ステータスコード含む）、exit code 1 |
| その他例外 | stderr にメッセージ、exit code 1 |

---

## Dependencies

| パッケージ | 用途 |
|---|---|
| `System.CommandLine` | CLIパーサー |
| `ClickUpClient`（プロジェクト参照） | APIクライアント本体 |

---

## .gitignore

リポジトリルートの `.gitignore`（または `src/ClickUpCli/.gitignore`）に追加:

```
src/ClickUpCli/config.json
```

---

## Out of Scope

- インタラクティブな操作
- タスクの作成・更新・削除
- ページネーション（page=0 のみ、ライブラリ側の方針と同じ）
- タイムトラッキング
