# show-config コマンド 設計ドキュメント

**日付:** 2026-05-06  
**ステータス:** 承認済み

## 概要

現在の設定内容をJSON形式で標準出力に表示する `show-config` コマンドを追加する。
AIエージェントや人間のオペレーターが、利用可能なlist名・IDを含む設定の現在値を素早く確認できるようにする。

## コマンド仕様

```
clickup show-config [--config <path>]
```

- 引数なし（グローバルフラグの `--config` のみ受け付ける）
- 設定が読み込めない場合は他コマンドと同じエラーを返して終了

## 出力フォーマット

```json
{
  "apiKey": "pk_****a1b2",
  "teamId": "12345678",
  "lists": {
    "my-list": "87654321",
    "another-list": "11223344"
  }
}
```

| フィールド | 型 | 説明 |
|---|---|---|
| `apiKey` | string | 末尾4文字を残して `****` でマスク。4文字以下は全て `****` |
| `teamId` | string | そのまま表示 |
| `lists` | object | 設定ファイルの `lists` マップをそのまま出力（名前→ID） |

## 実装

### 新規ファイル

**`cmd/clickup/show_config.go`**

- `newShowConfigCmd()` 関数でコブラコマンドを定義
- `loadConfig()` で設定を読み込み、`maskAPIKey()` でAPIキーをマスクしてから `printJSON()` で出力

### 変更ファイル

**`cmd/clickup/helpers.go`**

- `maskAPIKey(s string) string` を追加
  - `len(s) > 4` の場合: `"****" + s[len(s)-4:]`
  - それ以外: `"****"`

**`cmd/clickup/main.go`**

- `rootCmd.AddCommand(newShowConfigCmd())` を追加

## テスト方針

- `maskAPIKey` のユニットテストを `helpers_test.go`（または `show_config_test.go`）に追加
  - 通常のキー（5文字以上）
  - 4文字以下のキー
  - 空文字列
