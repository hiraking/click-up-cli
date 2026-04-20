# ClickUp CLI

ClickUp REST API v2 の薄い CLI ラッパー。AI エージェントやスクリプトから ClickUp タスクを JSON で取得するためのツール。

## セットアップ

### 1. ビルド・発行

```powershell
dotnet publish src/ClickUpCli/ClickUpCli.csproj -c Release -o out/clickup
```

### 2. 設定ファイルの作成

`src/ClickUpCli/config.json` を作成する（publish 時に自動で出力ディレクトリにコピーされる）。

```json
{
  "apiKey": "pk_YOUR_API_KEY_HERE",
  "teamId": "YOUR_TEAM_ID_HERE",
  "lists": {
    "work":      "LIST_ID_1",
    "study":     "LIST_ID_2"
  }
}
```

| フィールド | 説明 |
|---|---|
| `apiKey` | ClickUp の Personal API Token（Settings → Apps → API Token） |
| `teamId` | ワークスペース ID（URL の `/w/{teamId}/` から確認） |
| `lists` | リスト名 → リスト ID のマッピング。`--list` オプションで名前を指定するために使う |

> `config.json` は `.gitignore` で除外済み。コミットされない。

---

## コマンドリファレンス

### `get-tasks` — タスク一覧をツリー形式で取得

```
clickup get-tasks [options]
```

| オプション | 型 | 説明 |
|---|---|---|
| `--list <name>...` | string[] | 取得するリスト名（`config.json` の `lists` キー）。複数指定可。省略時は全リスト |
| `--status <name>...` | string[] | フィルタするステータス名。複数指定可（例: `"active"` `"in progress"`） |
| `--due-after <ISO8601>` | string | この日時より後の due_date を持つタスクに絞り込む |
| `--due-before <ISO8601>` | string | この日時より前の due_date を持つタスクに絞り込む |
| `--no-subtasks` | flag | サブタスクを取得しない（デフォルト: サブタスクあり） |

**出力:** ルートタスクの JSON 配列。サブタスクは各タスクの `subtasks` フィールドにネスト。

#### 使用例

```powershell
# 全リストのタスクを取得
clickup get-tasks

# work リストのタスクのみ
clickup get-tasks --list work

# work と study リストを同時に指定（2通りの書き方）
clickup get-tasks --list work study
clickup get-tasks --list work --list study

# ステータスでフィルタ
clickup get-tasks --list work --status active

# 今日中に期限が来るタスク
clickup get-tasks --due-before 2026-04-21T23:59:59+09:00

# 期限が過ぎたタスク（昨日以前）
clickup get-tasks --due-before 2026-04-20T00:00:00+09:00

# サブタスクなしで取得
clickup get-tasks --list work --no-subtasks
```

---

### `get-task` — 単一タスクを取得

```
clickup get-task <taskId>
```

| 引数 | 説明 |
|---|---|
| `taskId` | ClickUp タスク ID（必須） |

**出力:** 単一タスクの JSON オブジェクト。

#### 使用例

```powershell
clickup get-task 86exa7yq5
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
  "createdAt": "2026-04-19T15:09:41.393+00:00",
  "updatedAt": "2026-04-19T16:05:33.346+00:00",
  "subtasks": []
}
```

---

## エラーハンドリング

エラーは stderr に出力され、exit code 1 で終了する。

| ケース | メッセージ例 |
|---|---|
| `config.json` が見つからない | `Error: config.json not found at '...'` |
| 不明なリスト名 | `Error: Unknown list name 'foo'. Available: work, study, ...` |
| 日付フォーマット不正 | `Error: '--due-after' value '...' is not a valid ISO 8601 datetime.` |
| API エラー | `HTTP Error (404 NotFound): ...` |

---

## 注意事項

- ページネーションは page=0 のみ取得（大量タスクがある場合は絞り込みを使う）
- `--due-after` / `--due-before` フィルタは ClickUp API 側で処理される。親タスクが条件外でもサブタスクが条件にマッチする場合、そのサブタスクはルートタスクとして返る（`parentId` フィールドで親を確認できる）
