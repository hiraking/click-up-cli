---
name: clickup-cli-smoke-test
description: ClickUp CLI ツール（src/ClickUpCli）の動作確認手順。ビルド・発行後の疎通確認、コマンドオプションの動作検証、エラーハンドリングの確認を行う。ClickUpCli に変更を加えた後や初期セットアップ時に使用する。
---

# ClickUp CLI Smoke Test

リポジトリ: `C:\Users\平木大都\source\repos\playground\mine\click-up-client`  
publish 先: `out/clickup/`  
実行バイナリ: `out/clickup/clickup.exe`

## Step 1: Publish

```powershell
dotnet publish src/ClickUpCli/ClickUpCli.csproj -c Release -o out/clickup
```

期待: `Build succeeded` かつ `out/clickup/clickup.exe` が生成される。

## Step 2: config.json の確認

`src/ClickUpCli/config.json` が存在し、`apiKey` / `teamId` / `lists` がすべて設定済みであることを確認する。  
存在しない場合は `src/ClickUpCli/config.sample.json` をコピーして値を入力してから再 publish する（config.json は publish 時に自動コピーされる）。

## Step 3: エラーハンドリング確認

### 3-1: config.json なし → エラー終了

```powershell
Rename-Item out/clickup/config.json out/clickup/config.json.bak
out/clickup/clickup.exe get-tasks
$LASTEXITCODE
Rename-Item out/clickup/config.json.bak out/clickup/config.json
```

期待:
- stderr に `Error: config.json not found at '...'` が出力される
- exit code = 1

### 3-2: 不明なリスト名 → エラー終了

```powershell
out/clickup/clickup.exe get-tasks --list nonexistent
$LASTEXITCODE
```

期待:
- stderr に `Error: Unknown list name 'nonexistent'. Available: study, admin, ...` が出力される
- exit code = 1

## Step 4: get-tasks 動作確認

### 4-1: 全リスト取得

```powershell
out/clickup/clickup.exe get-tasks
```

期待:
- stdout に `TaskSummary[]` の JSON 配列が出力される（空配列 `[]` も正常）
- 日本語が `\uXXXX` ではなく文字そのままで出力される（例: `"名前"` ）
- exit code = 0

### 4-2: 単一リスト絞り込み

config.json に定義されているリスト名を1つ選んで実行する（例: `study`）。

```powershell
out/clickup/clickup.exe get-tasks --list study
```

期待: `listId` が `study` に対応する ID のタスクのみ返る。

### 4-3: 複数リスト指定

```powershell
out/clickup/clickup.exe get-tasks --list study work
```

期待: `study` と `work` 両方のタスクが返る。

### 4-4: --no-subtasks

```powershell
out/clickup/clickup.exe get-tasks --list study --no-subtasks
```

期待: 各タスクの `subtasks` が `[]` になる。

## Step 5: get-task 動作確認

Step 4-1 の出力から任意のタスク ID をコピーして実行する。

```powershell
out/clickup/clickup.exe get-task <taskId>
```

期待:
- stdout に単一 `TaskSummary` の JSON オブジェクトが出力される
- exit code = 0

## 合否判定

すべてのステップで期待通りの出力・exit code が確認できれば PASS。  
1つでも外れがあれば FAIL として原因を報告する。
