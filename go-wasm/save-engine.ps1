<#
.SYNOPSIS
  Build the UCI engine and save a timestamped copy to engines/.

.DESCRIPTION
  Compiles cmd/uci, writes the binary to engines/ named my-stockfish-YYYY-MM-DD-HHMM.exe,
  and prints the path so it can be copied into a cutechess-cli match command.

  Run from go-wasm/:
    ./save-engine.ps1
    ./save-engine.ps1 -Tag quiet        -> my-stockfish-2026-06-20-1430-quiet.exe

.OUTPUTS
  The full path of the saved binary.
#>
param(
  [string]$Tag = ""
)

$ErrorActionPreference = "Stop"

Write-Output "Building UCI engine..."
go build -o bin/my-stockfish.exe ./cmd/uci
if ($LASTEXITCODE -ne 0) {
  Write-Error "Build failed."
  exit 1
}

$stamp = Get-Date -Format "yyyy-MM-dd-HHmm"
$name = "my-stockfish-$stamp"
if ($Tag -ne "") { $name += "-$Tag" }
$name += ".exe"

$dest = Join-Path (Resolve-Path engines) $name
Copy-Item bin/my-stockfish.exe $dest

Write-Output "Saved: $dest"