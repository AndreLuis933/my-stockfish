param(
    [string]$Tag = ""
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$engineDir = Join-Path $PSScriptRoot "engines"
$binPath = Join-Path $PSScriptRoot "bin\my-stockfish.exe"

if (-not (Test-Path -LiteralPath $engineDir)) {
    New-Item -ItemType Directory -Path $engineDir | Out-Null
}

Write-Host "Building UCI engine..." -ForegroundColor Cyan
& go build -o $binPath ./cmd/uci
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed." -ForegroundColor Red
    exit 1
}

$stamp = Get-Date -Format "yyyy-MM-dd-HHmm"
$name = if ($Tag) { "my-stockfish-$stamp-$Tag.exe" } else { "my-stockfish-$stamp.exe" }
$dest = Join-Path $engineDir $name

Copy-Item -LiteralPath $binPath -Destination $dest
Write-Host "Saved: $dest" -ForegroundColor Green