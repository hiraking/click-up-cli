# ClickUp CLI

ClickUp REST API v2 の薄い CLI ラッパー。AI エージェントやスクリプトから ClickUp タスクを JSON で取得・作成するためのツール。

## セットアップ

### 1. インストール

```bash
go install ./cmd/clickup
```

`$GOPATH/bin`（通常 `~/go/bin`）に `clickup` バイナリが配置される。`$PATH` に含まれていればそのまま実行できる。

### 2. 設定ファイルの作成

`~/.clickup/config.json` を作成する（`config.sample.json` をコピーして編集）。

```json
{
  "apiKey": "pk_YOUR_API_KEY_HERE",
  "teamId": "YOUR_TEAM_ID_HERE",
  "lists": {
    "work":  "LIST_ID_1",
    "study": "LIST_ID_2"
  }
}
```

| フィールド | 説明 |
|---|---|
| `apiKey` | ClickUp の Personal API Token（Settings → Apps → API Token） |
| `teamId` | ワークスペース ID（URL の `/w/{teamId}/` から確認） |
| `lists` | リスト名 → リスト ID のマッピング。`--list` オプションで名前を指定するために使う |

> `config.json` はリポジトリ外（`~/.clickup/config.json`）に配置するため、コミットされない。

---

## コマンドリファレンス

### `get-tasks` — タスク一覧をツリー形式で取得

```
clickup get-tasks [options]
```

| オプション | 型 | 説明 |
|---|---|---|
| `--list <name>` | string | 取得するリスト名（`config.json` の `lists` キー）。複数指定可（`--list work --list study`）。省略時は全リスト |
| `--status <name>` | string | フィルタするステータス名。複数指定可 |
| `--due-after <ISO8601>` | string | この日時より後の due_date を持つタスクに絞り込む |
| `--due-before <ISO8601>` | string | この日時より前の due_date を持つタスクに絞り込む |
| `--no-subtasks` | flag | サブタスクを取得しない（デフォルト: サブタスクあり） |

**出力:** ルートタスクの JSON 配列。サブタスクは各タスクの `subtasks` フィールドにネスト。

> **日付のタイムゾーンについて:** オフセットなしで渡した場合（例: `"2026-05-01"` や `"2026-05-01T09:00"`）は **JST (+09:00)** として扱われる。オフセットを明示した場合（例: `"2026-05-01T00:00:00Z"` や `"2026-05-01T09:00:00+09:00"`）はその値をそのまま使用する。

#### 使用例

```bash
# 全リストのタスクを取得
clickup get-tasks

# work リストのタスクのみ
clickup get-tasks --list work

# work と study リストを指定
clickup get-tasks --list work --list study

# ステータスでフィルタ
clickup get-tasks --list work --status active

# 今日中に期限が来るタスク
clickup get-tasks --due-before 2026-04-21T23:59:59+09:00

# サブタスクなしで取得
clickup get-tasks --list work --no-subtasks
```

---

### `create-task` — タスクを新規作成

```
clickup create-task <name> --list <name> [options]
```

| 引数/オプション | 型 | 説明 |
|---|---|---|
| `name` | string | タスク名（必須） |
| `--list <name>` | string | 作成先リスト名（必須） |
| `--description <text>` | string | タスクの説明 |
| `--parent <taskId>` | string | 親タスク ID。指定するとサブタスクとして作成 |
| `--status <name>` | string | ステータス名（例: `"to do"`, `"in progress"`） |
| `--priority <value>` | string | 優先度: `urgent` / `high` / `normal` / `low` |
| `--due-date <ISO8601>` | string | 期日 |
| `--start-date <ISO8601>` | string | 開始日 |
| `--time-estimate <分>` | int | 見積もり時間（分単位） |

**出力:** 作成されたタスクの JSON オブジェクト。

#### 使用例

```bash
# 最小構成
clickup create-task "新しいタスク" --list work

# オプション全指定
clickup create-task "設計書を書く" --list work \
  --description "アーキテクチャ設計書の作成" \
  --parent "86exa7yq5" \
  --status "to do" \
  --priority high \
  --due-date "2026-05-01T18:00+09:00" \
  --start-date "2026-04-25T09:00" \
  --time-estimate 120
```

---

### `get-task` — 単一タスクを取得

```
clickup get-task <taskId>
```

#### 使用例

```bash
clickup get-task 86exa7yq5
```

---

### `update-task` — タスクを更新

```
clickup update-task <taskId> [options]
```

| 引数/オプション | 型 | 説明 |
|---|---|---|
| `taskId` | string | 更新対象のタスク ID（必須） |
| `--name <text>` | string | 新しいタスク名 |
| `--description <text>` | string | 新しい説明 |
| `--status <name>` | string | 新しいステータス名（例: `"to do"`, `"in progress"`） |
| `--priority <value>` | string | 新しい優先度: `urgent` / `high` / `normal` / `low` |
| `--due-date <ISO8601>` | string | 新しい期日 |
| `--start-date <ISO8601>` | string | 新しい開始日 |
| `--time-estimate <分>` | int | 新しい見積もり時間（分単位） |
| `--parent <taskId>` | string | 新しい親タスク ID |
| `--clear <field>` | string | フィールドをクリアする（繰り返し可） |

指定したオプションのフィールドのみ更新される。未指定のフィールドは変更されない。

**出力:** 更新後のタスクの JSON オブジェクト。

#### `--clear` でクリアできるフィールド

| フィールド名 | 説明 |
|---|---|
| `description` | 説明をクリア |
| `status` | ステータスをクリア |
| `priority` | 優先度をクリア |
| `due-date` | 期日をクリア |
| `start-date` | 開始日をクリア |
| `time-estimate` | 見積もり時間をクリア |

> `name` はクリア不可（ClickUp API の必須フィールド）。  
> `parent` はクリア不可（ClickUp API がサブタスクの親削除を非サポート）。

#### 使用例

```bash
# タスク名を変更する
clickup update-task 86exa7yq5 --name "新しいタスク名"

# ステータスと優先度を同時に変更する
clickup update-task 86exa7yq5 --status "in progress" --priority high

# 期日をクリアする
clickup update-task 86exa7yq5 --clear due-date

# 名前を変更しつつ説明をクリアする
clickup update-task 86exa7yq5 --name "新しい名前" --clear description

# 複数フィールドをクリアする
clickup update-task 86exa7yq5 --clear due-date --clear priority
```

---

## 出力フォーマット

`TaskSummary` の camelCase JSON。

```json
{
  "id": "86exa7yq5",
  "name": "英語学習",
  "status": "active",
  "priority": null,
  "parentId": null,
  "url": "https://app.clickup.com/t/86exa7yq5",
  "dueDate": null,
  "description": "",
  "listId": "901817486451",
  "listName": "学習",
  "createdAt": "2026-04-19T15:09:41.393Z",
  "updatedAt": "2026-04-19T16:05:33.346Z",
  "subtasks": []
}
```

---

## エラーハンドリング

エラーは stderr に出力され、exit code 1 で終了する。

| ケース | メッセージ例 |
|---|---|
| `config.json` が見つからない | `config file not found: ...` |
| 不明なリスト名 | `Error: Unknown list name 'foo'. Available: work, study` |
| 日付フォーマット不正 | `Error: '--due-after' value '...' is not a valid ISO 8601 datetime.` |
| 不正な優先度 | `Error: Invalid priority 'foo'. Use urgent, high, normal, or low.` |
| API エラー | `HTTP Error (404): ...` |

---

## 注意事項

- タスク取得は最大 10 ページ（最大 1,000 件）まで自動ページネーション。1,000 件を超える場合は警告を stderr に出力し、取得済み分を返す
- `--due-after` / `--due-before` フィルタは ClickUp API 側で処理される
- `--list` は複数回指定可能（`--list work --list study`）
