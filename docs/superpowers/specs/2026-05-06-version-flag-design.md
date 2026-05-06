---
title: --version フラグの追加
date: 2026-05-06
status: approved
---

## Overview

`clickup --version` でバージョン番号を表示できるようにする。cobra の組み込み `Version` フィールドと Go の `-ldflags` によるビルド時埋め込みを組み合わせる。

## Goals

- `clickup --version` および `clickup version` でバージョン番号を表示する
- リリースビルドでは git タグと一致したバージョン番号が表示される
- ローカルビルド（ldflags なし）では `dev` と表示される

## Non-Goals

- ビルド日時・コミットハッシュの表示
- カスタム出力フォーマット

## Design

### `cmd/clickup/main.go`

パッケージレベル変数 `version` を追加し、`rootCmd.Version` に設定する。

```go
var version = "dev"

func main() {
    rootCmd := &cobra.Command{
        Use:           "clickup",
        Short:         "ClickUp API CLI wrapper",
        Version:       version,
        SilenceUsage:  true,
        SilenceErrors: true,
    }
    // ...
}
```

cobra が `--version` フラグと `version` サブコマンドを自動登録する。

### `.github/skills/create-github-release/release.ps1`

`go build` の呼び出しに `-ldflags` を追加する。

```powershell
go build -ldflags "-X main.version=$Version" -o $t.Out $pkg
```

## 動作

| コマンド | 出力 |
|---|---|
| `clickup --version` | `clickup version v0.3.0` |
| `clickup version` | `clickup version v0.3.0` |
| ローカルビルド（ldflags なし） | `clickup version dev` |

## 変更ファイル

| ファイル | 変更内容 |
|---|---|
| `cmd/clickup/main.go` | `var version = "dev"` 追加、`rootCmd.Version = version` 追加 |
| `.github/skills/create-github-release/release.ps1` | `go build` に `-ldflags "-X main.version=$Version"` 追加 |

## Testing

cobra の Version 機能はライブラリ側でテスト済みのため、アプリ側の単体テストは不要。リリース後に `clickup --version` で手動確認する。
