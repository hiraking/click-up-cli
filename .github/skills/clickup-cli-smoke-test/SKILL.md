---
name: clickup-cli-smoke-test
description: ClickUp CLI ツール（cmd/clickup）の動作確認手順。ビルド後の疎通確認、コマンドオプションの動作検証、エラーハンドリングの確認を行う。CLI に変更を加えた後や初期セットアップ時に使用する。
---

# ClickUp CLI Smoke Test

リポジトリ: `C:\Users\平木大都\source\repos\playground\mine\click-up-client`  
ビルド出力先: `out\clickup.exe`  
実行バイナリ: `out\clickup.exe`

## Step 1: ビルド

```powershell
go build -o out\clickup.exe .\cmd\clickup\
```

期待: エラーなしで `out\clickup.exe` が生成される。

## Step 2: config.json の確認

`out\config.json` が存在し、`apiKey` / `teamId` / `lists` がすべて設定済みであることを確認する。  
存在しない場合は `config.sample.json` をコピーして値を入力してから `out\` に配置する。

```powershell
Copy-Item config.sample.json out\config.json
# エディタで out\config.json を開いて apiKey / teamId / lists を設定する
```

> **注意**: `config.json` はバイナリと同じディレクトリ（`out\`）に配置する必要がある。  
> `go run` では `os.Executable` の解決先が異なるため、必ずビルド済みバイナリを使用する。

## Step 3: エラーハンドリング確認

### 3-1: config.json なし → エラー終了

```powershell
Rename-Item out\config.json out\config.json.bak
out\clickup.exe get-tasks
$LASTEXITCODE
Rename-Item out\config.json.bak out\config.json
```

期待:
- stderr に `Error: config.json not found at '...'` が出力される
- exit code = 1

### 3-2: 不明なリスト名 → エラー終了

```powershell
out\clickup.exe get-tasks --list nonexistent
$LASTEXITCODE
```

期待:
- stderr に `Error: Unknown list name 'nonexistent'. Available: ...` が出力される
- exit code = 1

### 3-3: create-task で --list 省略 → エラー終了

```powershell
out\clickup.exe create-task --name "テストタスク"
$LASTEXITCODE
```

期待:
- stderr に `required flag(s) "list" not set` が出力される
- exit code = 1

## Step 4: get-tasks 動作確認

### 4-1: 全リスト取得

```powershell
out\clickup.exe get-tasks
```

期待:
- stdout に `TaskSummary` の JSON 配列が出力される（空配列 `[]` も正常）
- 日本語が `\uXXXX` ではなく文字そのままで出力される（例: `"名前"` ）
- 各タスクに `"subtasks": []` が含まれる（omitempty なし）
- exit code = 0

### 4-2: 単一リスト絞り込み

config.json に定義されているリスト名を1つ選んで実行する（例: `study`）。

```powershell
out\clickup.exe get-tasks --list study
```

期待: `listId` が `study` に対応する ID のタスクのみ返る。

### 4-3: 複数リスト指定

```powershell
out\clickup.exe get-tasks --list study --list work
```

期待: `study` と `work` 両方のタスクが返る。

### 4-4: --no-subtasks

```powershell
out\clickup.exe get-tasks --list study --no-subtasks
```

期待: 各タスクの `subtasks` が `[]` になる（ネストなし）。

### 4-5: 期日フィルタ

```powershell
out\clickup.exe get-tasks --due-before 2026-12-31
```

期待: `dueDate` が 2026-12-31 より前のタスクのみ返る。

## Step 5: get-task 動作確認

Step 4-1 の出力から任意のタスク ID をコピーして実行する。

```powershell
out\clickup.exe get-task <taskId>
```

期待:
- stdout に単一 `TaskSummary` の JSON オブジェクトが出力される
- `startDate` フィールドが存在する場合は ISO 8601 形式で出力される
- exit code = 0

## Step 6: create-task 動作確認

### 6-1: 最小構成（名前のみ）

config.json に定義されているリスト名を1つ選んで実行する（例: `study`）。

```powershell
out\clickup.exe create-task --list study --name "スモークテスト用タスク"
```

期待:
- stdout に作成された `TaskSummary` の JSON オブジェクトが出力される
- `"name": "スモークテスト用タスク"` が含まれる
- exit code = 0

### 6-2: オプション付き

```powershell
out\clickup.exe create-task --list study --name "期日付きタスク" --due-date 2026-12-31 --priority normal
```

期待:
- `"dueDate"` が出力に含まれる
- `"priority": "normal"` が含まれる
- exit code = 0

### 6-3: start-date 付き（読み取り確認）

```powershell
$result = out\clickup.exe create-task --list study --name "開始日付きタスク" --start-date 2026-06-01 | ConvertFrom-Json
out\clickup.exe get-task $result.id
```

期待: `get-task` の出力に `"startDate"` フィールドが含まれる。

## 合否判定

すべてのステップで期待通りの出力・exit code が確認できれば PASS。  
1つでも外れがあれば FAIL として原因を報告する。
