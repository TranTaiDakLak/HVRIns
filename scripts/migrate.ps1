#requires -Version 5.1
<#
.SYNOPSIS
    One-time Go source text migration for the HVRIns Instagram clone.

.DESCRIPTION
    Applies the pure, idempotent Go text-transformation rules (G1-G4) from the
    hvrins-instagram-clone spec to every *.go file under a root path.

    Rule order matters: G1 runs before G2 so the more specific facebook-subpath
    import prefix is rewritten before the general HVR/ prefix.

        G1  "HVR/internal/facebook   ->  "HVRIns/internal/instagram   (literal)
        G2  "HVR/                     ->  "HVRIns/                     (literal, remaining)
        G3  \bpackage facebook\b      ->  package instagram            (regex, word boundary)
        G4  \bfacebook\.([A-Z])       ->  instagram.$1                 (regex, uppercase guard)

    INVARIANT: facebook.com is never modified. G4 only matches facebook. followed
    by an uppercase ASCII letter [A-Z], and the 'c' in '.com' is lowercase, so URL
    literals such as https://www.facebook.com/... are preserved byte-for-byte.

    The transform is idempotent: after a successful run no "HVR/ import prefix,
    no `package facebook` declaration, and no `facebook.<Upper>` qualifier remains,
    so a second run produces no further changes.

    NOTE: The real source Go module is `HVR` (not `HVRFb`); HVR/ prefixes are used.

.PARAMETER Root
    Root directory to scan recursively for *.go files. Defaults to e:\WEMAKE\HVRIns.

.EXAMPLE
    pwsh -File scripts\migrate.ps1
    pwsh -File scripts\migrate.ps1 -Root e:\WEMAKE\HVRIns
#>
[CmdletBinding()]
param(
    [string]$Root = 'e:\WEMAKE\HVRIns'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Pure (string) -> (string) transform applying rules G1-G4 in order.
function Convert-GoSource {
    param([Parameter(Mandatory)][string]$Text)

    # G1: specific facebook-subpath import prefix (literal). Must run before G2.
    $out = $Text.Replace('"HVR/internal/facebook', '"HVRIns/internal/instagram')

    # G2: any remaining HVR/ import prefix (literal).
    $out = $out.Replace('"HVR/', '"HVRIns/')

    # G3: package declaration, word-boundary guarded.
    $out = [regex]::Replace($out, '\bpackage facebook\b', 'package instagram')

    # G4: symbol qualifier only - facebook. followed by an uppercase ASCII letter.
    #     The uppercase guard preserves facebook.com (lowercase 'c').
    $out = [regex]::Replace($out, '\bfacebook\.([A-Z])', 'instagram.$1')

    return $out
}

function Invoke-GoMigration {
    param([Parameter(Mandatory)][string]$Root)

    $resolvedRoot = (Resolve-Path -LiteralPath $Root).Path
    Write-Host "Applying Go transforms (G1-G4) under: $resolvedRoot"

    # UTF-8 without BOM, matching Go source conventions.
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)

    # Recursively find all *.go files, excluding anything under a .kiro/ directory.
    $goFiles = Get-ChildItem -LiteralPath $resolvedRoot -Recurse -File -Filter '*.go' |
        Where-Object { $_.FullName -notmatch '(\\|/)\.kiro(\\|/)' }

    $changedCount = 0
    foreach ($file in $goFiles) {
        $original = [System.IO.File]::ReadAllText($file.FullName)
        $transformed = Convert-GoSource -Text $original

        if ($transformed -ne $original) {
            [System.IO.File]::WriteAllText($file.FullName, $transformed, $utf8NoBom)
            Write-Host "modified: $($file.FullName)"
            $changedCount++
        }
    }

    Write-Host "Done. Modified $changedCount file(s) of $($goFiles.Count) scanned."
    return $changedCount
}

# Only auto-run when invoked as a script (not when dot-sourced for testing).
if ($MyInvocation.InvocationName -ne '.') {
    Invoke-GoMigration -Root $Root | Out-Null
}
