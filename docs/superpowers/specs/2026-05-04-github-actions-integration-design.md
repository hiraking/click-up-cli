---
title: GitHub Actions Integration — Environment Variable & Config Path Support
date: 2026-05-04
status: approved
---

## Overview

別リポジトリの GitHub Actions ワークフローから `clickup` CLI を実行できるようにする。
`apiKey` / `teamId` は GitHub Secrets 経由で環境変数として渡し、`lists` はオプションで設定ファイル経由で渡す。

## Goals

- `CLICKUP_API_KEY` / `CLICKUP_TEAM_ID` 環境変数で apiKey / teamId を指定できる
- `--config` フラグまたは `CLICKUP_CONFIG` 環境変数で設定ファイルパスを指定できる
- デフォルトパス (`~/.clickup/config.json`) が存在しない場合でも、環境変数だけで動作する
- 既存のローカル利用（`~/.clickup/config.json` に全フィールドを書く）は変更なし

## Non-Goals

- `lists` の環境変数サポート（設定ファイル経由のみ）
- GitHub Actions の workflow ファイル自体の提供

## Config Loading Flow

### 設定ファイルパスの決定（`helpers.go`）

優先順位（高い順）:

1. `--config` フラグ（明示的・ファイル必須）
2. `CLICKUP_CONFIG` 環境変数（明示的・ファイル必須）
3. `~/.clickup/config.json`（デフォルト・任意）
   - ファイルが存在しない場合は無視し、空パス `""` で `config.Load()` を呼ぶ

### apiKey / teamId の値の決定（`config.Load()`）

優先順位（高い順）:

1. `CLICKUP_API_KEY` / `CLICKUP_TEAM_ID` 環境変数
2. 設定ファイルの値
3. どちらもなければバリデーションエラー

### lists

設定ファイルからのみ読み込む。環境変数での指定は非サポート。

## Error Handling

| 状況 | 挙動 |
|---|---|
| `--config` 指定 → ファイルなし | エラー |
| `CLICKUP_CONFIG` 指定 → ファイルなし | エラー |
| デフォルトパス → ファイルなし → env var なし | `config: apiKey is required`（既存と同じ） |
| デフォルトパス → ファイルなし → env var あり | 正常動作 |
| ファイルに `apiKey` あり + env var あり | env var が優先 |

## Code Changes

変更対象は3ファイルのみ。サブコマンド側は変更なし。

### `internal/config/config.go`

`Load(path string)` を拡張:
- `path == ""` → ファイル読み込みをスキップ（env var のみモード）
- `path != ""` → 現状通りファイル読み込み
- ファイル読み込み後（またはスキップ後）に `CLICKUP_API_KEY` / `CLICKUP_TEAM_ID` でオーバーライド
- バリデーションは変わらず

### `cmd/clickup/main.go`

rootCmd に persistent flag を追加:

```go
var configPath string
rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path")
```

### `cmd/clickup/helpers.go`

`loadConfig()` を修正:

```
path 決定:
  1. configPath フラグ変数 → あればそのまま使用
  2. CLICKUP_CONFIG env var → あればそのまま使用
  3. ~/. clickup/config.json を stat チェック
     → 存在すれば使用、存在しなければ "" を渡す

→ config.Load(path) を呼ぶ
```

## Usage in GitHub Actions

```yaml
# time-report のみ（設定ファイル不要）
- run: clickup time-report --start "2026-05-01T00:00:00+09:00" --end "2026-05-08T00:00:00+09:00"
  env:
    CLICKUP_API_KEY: ${{ secrets.CLICKUP_API_KEY }}
    CLICKUP_TEAM_ID: ${{ secrets.CLICKUP_TEAM_ID }}

# lists も使う場合（設定ファイルをその場で作成）
- run: |
    echo '{"lists":{"work":"123456"}}' > /tmp/clickup.json
    clickup --config /tmp/clickup.json get-tasks
  env:
    CLICKUP_API_KEY: ${{ secrets.CLICKUP_API_KEY }}
    CLICKUP_TEAM_ID: ${{ secrets.CLICKUP_TEAM_ID }}
```

## Backward Compatibility

ローカル利用（`~/.clickup/config.json` に全フィールドを書く従来の使い方）はそのまま動作する。変更は加法的で既存の動作を壊さない。
