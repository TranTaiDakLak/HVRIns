<#
.SYNOPSIS
    Identity-rename precondition guard for the HVRIns migration (task 2.2, Requirement 1.5).

.DESCRIPTION
    Formalizes the go.mod / wails.json identity rename (originally applied by task 2.1)
    as a reusable, re-runnable migration helper.

    Behavior:
      - Before writing anything, the script verifies that both `go.mod` and `wails.json`
        exist and are writable. If EITHER file is missing or unwritable, the script aborts
        the entire rename, writes NOTHING, leaves all target files unchanged, and emits an
        error naming the affected file (Requirement 1.5).
      - When the guards pass, it sets the `go.mod` module line to `module HVRIns`, and the
        `wails.json` `name` / `outputfilename` fields to `HVRIns`.
      - The operation is idempotent: if a value is already correct it is left untouched and
        reported as "already correct". A pre-existing `module HVR` or `module HVRIns` line is
        accepted as a matching source.

.PARAMETER Root
    The project root containing go.mod and wails.json. Defaults to e:\WEMAKE\HVRIns.

.OUTPUTS
    Prints a per-file summary of what changed (or that values were already correct).
    Exits with code 0 on success, 1 on a guard failure / abort.
#>
[CmdletBinding()]
param(
    [string]$Root = 'e:\WEMAKE\HVRIns'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$TargetModule = 'HVRIns'
$TargetName   = 'HVRIns'

$goModPath    = Join-Path $Root 'go.mod'
$wailsJsonPath = Join-Path $Root 'wails.json'

# ----------------------------------------------------------------------------
# Helper: test whether an existing file is writable (can be opened for write).
# ----------------------------------------------------------------------------
function Test-Writable {
    param([string]$Path)
    try {
        $fs = [System.IO.File]::Open($Path, [System.IO.FileMode]::Open, [System.IO.FileAccess]::Write, [System.IO.FileShare]::ReadWrite)
        $fs.Close()
        $fs.Dispose()
        return $true
    } catch {
        return $false
    }
}

# ----------------------------------------------------------------------------
# Guard phase: stat + writability check for BOTH files BEFORE any write.
# Abort (write nothing) if any guard fails.
# ----------------------------------------------------------------------------
$guardErrors = @()

foreach ($f in @($goModPath, $wailsJsonPath)) {
    if (-not (Test-Path -LiteralPath $f -PathType Leaf)) {
        $guardErrors += "Required file is missing: $f"
        continue
    }
    if (-not (Test-Writable -Path $f)) {
        $guardErrors += "Required file is not writable: $f"
    }
}

if ($guardErrors.Count -gt 0) {
    foreach ($e in $guardErrors) {
        Write-Error $e -ErrorAction Continue
    }
    Write-Error 'Identity rename aborted: precondition guard failed. No files were changed.' -ErrorAction Continue
    exit 1
}

# ----------------------------------------------------------------------------
# go.mod: set the module line to `module HVRIns` (idempotent).
# ----------------------------------------------------------------------------
$goModText = Get-Content -LiteralPath $goModPath -Raw
$goModNew  = [regex]::Replace($goModText, '(?m)^\s*module\s+\S+', "module $TargetModule")

if ($goModNew -ne $goModText) {
    Set-Content -LiteralPath $goModPath -Value $goModNew -NoNewline
    Write-Output "go.mod: module line set to 'module $TargetModule'."
} else {
    Write-Output "go.mod: module already correct ('module $TargetModule') - no change."
}

# ----------------------------------------------------------------------------
# wails.json: set name + outputfilename to HVRIns (idempotent).
# ----------------------------------------------------------------------------
$wailsText = Get-Content -LiteralPath $wailsJsonPath -Raw
$wailsNew  = $wailsText
# Anchor to top-level keys (2-space indentation in the Wails-generated file) so the
# nested `author.name` (4-space indentation) is never matched.
$wailsNew  = [regex]::Replace($wailsNew, '(?m)^(  "name"\s*:\s*")[^"]*(")',           "`${1}$TargetName`${2}")
$wailsNew  = [regex]::Replace($wailsNew, '(?m)^(  "outputfilename"\s*:\s*")[^"]*(")', "`${1}$TargetName`${2}")

if ($wailsNew -ne $wailsText) {
    Set-Content -LiteralPath $wailsJsonPath -Value $wailsNew -NoNewline
    Write-Output "wails.json: name/outputfilename set to '$TargetName'."
} else {
    Write-Output "wails.json: name/outputfilename already correct ('$TargetName') - no change."
}

Write-Output 'Identity rename complete.'
exit 0
