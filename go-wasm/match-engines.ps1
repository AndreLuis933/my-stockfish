<#
.SYNOPSIS
  Run a cutechess-cli match between two saved engine binaries.

.DESCRIPTION
  Accepts two .exe paths (from engines/) and runs a cutechess-cli match with
  sensible defaults: 1+0.1 tc, EPD openings, draw adjudication, PGN output.
  All cutechess parameters can be overridden via the remaining args.

  Run from go-wasm/:
    ./match-engines.ps1 engines/my-stockfish-2026-06-20-1430.exe engines/my-stockfish-2026-06-20-1600.exe
    ./match-engines.ps1 old.exe new.exe -rounds 100 -concurrency 16
    ./match-engines.ps1 old.exe new.exe -each proto=uci tc=5+0.5

.PARAMETER Old
  Path to the baseline engine .exe.

.PARAMETER New
  Path to the candidate engine .exe.
#>
param(
  [Parameter(Mandatory = $true, Position = 0)][string]$Old,
  [Parameter(Mandatory = $true, Position = 1)][string]$New,
  [Parameter(ValueFromRemainingArguments = $true)][string[]]$ExtraArgs
)

$ErrorActionPreference = "Stop"

$oldAbs = (Resolve-Path $Old).Path
$newAbs = (Resolve-Path $New).Path
$oldName = [IO.Path]::GetFileNameWithoutExtension($oldAbs)
$newName = [IO.Path]::GetFileNameWithoutExtension($newAbs)

$stamp = Get-Date -Format "yyyy-MM-dd-HHmm"
$pgn = Join-Path (Resolve-Path engines) "match-${oldName}_vs_${newName}-$stamp.pgn"

$openings = Join-Path (Resolve-Path bin) "openings16.epd"
if (-not (Test-Path $openings)) { $openings = $null }

# Defaults the user can override via ExtraArgs. We detect overrides by key and
# drop the default entry when the user supplied the same key.
$defaults = [ordered]@{
  "-each"       = @("proto=uci", "tc=1+0.1")
  "-rounds"     = @("40")
  "-concurrency"= @("8")
  "-draw"       = @("movenumber=30", "movecount=8", "score=20")
  "-pgnout"     = @($pgn)
}
if ($openings) { $defaults["-openings"] = @("file=$openings", "format=epd") }

$overrideKeys = @()
for ($i = 0; $i -lt $ExtraArgs.Count; $i++) {
  if ($ExtraArgs[$i] -match '^-[a-z]') { $overrideKeys += $ExtraArgs[$i] }
}

$built = @(
  "-engine", "name=$oldName", "cmd=`"$oldAbs`"",
  "-engine", "name=$newName", "cmd=`"$newAbs`""
)
foreach ($key in $defaults.Keys) {
  if ($key -in $overrideKeys) { continue }
  $built += @($key) + $defaults[$key]
}
$allArgs = $built + $ExtraArgs

Write-Output "Match: $oldName  vs  $newName"
Write-Output "PGN:   $pgn"
Write-Output ""

& cutechess-cli @allArgs