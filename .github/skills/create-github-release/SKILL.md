---
name: create-github-release
description: Use when creating a new GitHub Release for click-up-cli. Covers version number decision, release note generation from git log, multi-platform build, tag creation, and asset upload.
---

# Create GitHub Release

リリーススクリプト: `.github/skills/create-github-release/release.ps1`

## Step 1: バージョン番号を決定する

ユーザーからバージョン番号を受け取る。指示がなければ以下を確認して提案する。

```powershell
# 最新タグを確認
git tag --sort=-version:refname | Select-Object -First 1
```

セマンティックバージョニング (`vX.Y.Z`) に従う。
- バグ修正のみ → パッチ番号をインクリメント (例: v0.1.0 → v0.1.1)
- 後方互換の新機能 → マイナー番号をインクリメント (例: v0.1.0 → v0.2.0)
- 破壊的変更 → メジャー番号をインクリメント (例: v0.1.0 → v1.0.0)

ユーザーが判断できない場合は変更内容を確認してから提案する。

## Step 2: リリースノートを生成する

前回タグからの差分コミットをもとにリリースノートを作成する。

```powershell
# 前回タグ以降のコミット一覧
$prevTag = git tag --sort=-version:refname | Select-Object -First 1
git log "$prevTag..HEAD" --oneline
```

コミットメッセージをもとに以下の形式でリリースノートの草案を作成し、ユーザーに確認を求める。

```
## 変更内容

### 新機能
- ...

### バグ修正
- ...

### その他
- ...
```

ユーザーが承認したらStep 3へ進む。

## Step 3: スクリプトを実行する

パラメータを確認してからスクリプトを実行する。ユーザーの承認を得てから実行すること。

```powershell
# 例
.\.github\skills\create-github-release\release.ps1 `
  -Version "v0.2.0" `
  -Title "v0.2.0 - ..." `
  -Notes @"
## 変更内容

### 新機能
- ...
"@
```

スクリプトが行うこと:
1. `git tag` でタグを作成し `origin` へプッシュ
2. Windows / Linux / macOS (amd64, arm64) 向けにバイナリをクロスビルド
3. `gh release create` でリリースを作成し `dist/` 配下のバイナリを添付

## 注意事項

- `gh` コマンドがインストール済みかつ認証済みであること (`gh auth status`)
- タグはスクリプト内で作成するため、事前に手動で作成しないこと
- 同一バージョンのタグが既に存在する場合はスクリプトがエラーになる
