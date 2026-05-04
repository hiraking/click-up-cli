param(
    [Parameter(Mandatory)][string]$Version,
    [Parameter(Mandatory)][string]$Title,
    [Parameter(Mandatory)][string]$Notes
)

$ErrorActionPreference = "Stop"

# バージョン形式チェック
if ($Version -notmatch '^v\d+\.\d+\.\d+$') {
    Write-Error "Version must be in format vX.Y.Z (e.g. v1.2.3)"
    exit 1
}

$distDir = "dist"
$pkg = "./cmd/clickup"

$targets = @(
    @{ OS = "windows"; Arch = "amd64"; Out = "$distDir/clickup-windows-amd64.exe" },
    @{ OS = "linux";   Arch = "amd64"; Out = "$distDir/clickup-linux-amd64"       },
    @{ OS = "darwin";  Arch = "amd64"; Out = "$distDir/clickup-darwin-amd64"      },
    @{ OS = "darwin";  Arch = "arm64"; Out = "$distDir/clickup-darwin-arm64"      }
)

# --- タグ作成 ---
Write-Host "==> Creating tag $Version"
git tag $Version -m $Title
git push origin $Version

# --- ビルド ---
if (Test-Path $distDir) { Remove-Item "$distDir\*" -Force }
else { New-Item -ItemType Directory $distDir | Out-Null }

foreach ($t in $targets) {
    Write-Host "==> Building $($t.Out)"
    $env:GOOS   = $t.OS
    $env:GOARCH = $t.Arch
    go build -o $t.Out $pkg
}

Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue

# --- リリース作成 ---
$assets = $targets | ForEach-Object { $_.Out }

Write-Host "==> Creating GitHub release $Version"
gh release create $Version @assets --title $Title --notes $Notes

Write-Host "Done: https://github.com/hiraking/click-up-cli/releases/tag/$Version"
