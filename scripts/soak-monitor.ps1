# soak-monitor.ps1 - Record HaVu RAM/CPU/Goroutine metrics every N seconds to CSV.
#
# Usage:
#   .\scripts\soak-monitor.ps1 -DurationMinutes 30 -OutputCsv .\soak.csv
#
# Requirements:
#   - HaVu.exe is running
#   - For NumGoroutine column: app started with DEBUG_PPROF=1 (port 6060)
#
# Output CSV columns:
#   Timestamp, RssMB, CpuPct, NumGoroutine, ThreadCount, HandleCount
#
# This script is READ-ONLY - does not modify app, does not inject code.

[CmdletBinding()]
param(
    [string]$ProcessName = "HaVu",
    [int]$PProfPort = 6060,
    [string]$OutputCsv = ".\soak-$(Get-Date -Format 'yyyyMMdd-HHmmss').csv",
    [int]$IntervalSec = 60,
    [int]$DurationMinutes = 60
)

$ErrorActionPreference = "Stop"

# Validate process exists
$proc = Get-Process -Name $ProcessName -ErrorAction SilentlyContinue
if (-not $proc) {
    Write-Error "Process '$ProcessName' is not running. Start app before monitoring."
    exit 1
}

# If multiple instances, take first one
if ($proc -is [array]) {
    $proc = $proc[0]
    Write-Warning "Multiple '$ProcessName' processes - monitoring PID $($proc.Id)"
}

Write-Host "=== Soak Monitor ===" -ForegroundColor Cyan
Write-Host "Process     : $ProcessName (PID $($proc.Id))"
Write-Host "Output CSV  : $OutputCsv"
Write-Host "Interval    : $IntervalSec seconds"
Write-Host "Duration    : $DurationMinutes minutes"
Write-Host "PProf port  : $PProfPort (need DEBUG_PPROF=1 for goroutine count)"
Write-Host ""

# Test pprof connectivity once at start
$pprofUrl = "http://127.0.0.1:$PProfPort/debug/pprof/goroutine?debug=1"
$pprofAvailable = $false
try {
    $null = Invoke-WebRequest -Uri $pprofUrl -TimeoutSec 3 -UseBasicParsing
    $pprofAvailable = $true
    Write-Host "[OK] pprof reachable - will record goroutine count" -ForegroundColor Green
} catch {
    Write-Host "[WARN] pprof not reachable - NumGoroutine will be N/A" -ForegroundColor Yellow
    Write-Host "       (Start app with DEBUG_PPROF=1 to enable)" -ForegroundColor Yellow
}
Write-Host ""

# CSV header
"Timestamp,RssMB,CpuPct,NumGoroutine,ThreadCount,HandleCount" | Out-File -FilePath $OutputCsv -Encoding utf8

# CPU tracker (delta between samples)
$lastCpuTime = $proc.TotalProcessorTime.TotalSeconds
$lastSampleAt = Get-Date

$endAt = (Get-Date).AddMinutes($DurationMinutes)
$sampleCount = 0

Write-Host "Starting - Ctrl+C to stop early." -ForegroundColor Cyan
Write-Host ""

while ((Get-Date) -lt $endAt) {
    Start-Sleep -Seconds $IntervalSec
    $sampleCount++

    try {
        # Refresh process info
        $proc = Get-Process -Id $proc.Id -ErrorAction Stop

        $now = Get-Date
        $rssMB = [math]::Round($proc.WorkingSet64 / 1MB, 1)

        # CPU% delta (kernel+user time / elapsed) - divide by cores to match Task Manager
        $curCpuTime = $proc.TotalProcessorTime.TotalSeconds
        $elapsed = ($now - $lastSampleAt).TotalSeconds
        $cpuPct = 0.0
        if ($elapsed -gt 0) {
            $cpuDelta = $curCpuTime - $lastCpuTime
            $cpuPct = [math]::Round(($cpuDelta / $elapsed) * 100 / [Environment]::ProcessorCount, 1)
        }
        $lastCpuTime = $curCpuTime
        $lastSampleAt = $now

        $threadCount = $proc.Threads.Count
        $handleCount = $proc.HandleCount

        # NumGoroutine from pprof text format: "goroutine profile: total N"
        $numGoroutine = "N/A"
        if ($pprofAvailable) {
            try {
                $resp = Invoke-WebRequest -Uri $pprofUrl -TimeoutSec 5 -UseBasicParsing
                if ($resp.Content -match "goroutine profile: total (\d+)") {
                    $numGoroutine = $Matches[1]
                }
            } catch {
                # pprof temporarily unavailable - record N/A and continue
            }
        }

        $tsStr = $now.ToString("yyyy-MM-dd HH:mm:ss")
        $line = "$tsStr,$rssMB,$cpuPct,$numGoroutine,$threadCount,$handleCount"
        $line | Out-File -FilePath $OutputCsv -Encoding utf8 -Append

        Write-Host ("[{0}] RSS={1}MB CPU={2}% Goroutine={3} Threads={4}" -f `
            $tsStr, $rssMB, $cpuPct, $numGoroutine, $threadCount)
    } catch {
        Write-Warning "Sample error (process may have exited): $_"
        if (-not (Get-Process -Id $proc.Id -ErrorAction SilentlyContinue)) {
            Write-Host "Process exited - stopping monitor." -ForegroundColor Yellow
            break
        }
    }
}

Write-Host ""
Write-Host "=== Done ===" -ForegroundColor Cyan
Write-Host "Recorded $sampleCount samples to $OutputCsv"
Write-Host ""

# Quick summary
Write-Host "Summary:" -ForegroundColor Cyan
$rows = Import-Csv $OutputCsv
if ($rows.Count -gt 0) {
    $rssValues = $rows | ForEach-Object { [double]$_.RssMB }
    $cpuValues = $rows | ForEach-Object { [double]$_.CpuPct }
    $goroValues = $rows | Where-Object { $_.NumGoroutine -ne "N/A" } | ForEach-Object { [int]$_.NumGoroutine }

    Write-Host ("  RSS    start={0:N1}MB  end={1:N1}MB  peak={2:N1}MB  delta={3:N1}MB" -f `
        $rssValues[0], $rssValues[-1], `
        ($rssValues | Measure-Object -Maximum).Maximum, `
        ($rssValues[-1] - $rssValues[0]))
    Write-Host ("  CPU    avg={0:N1}%  peak={1:N1}%" -f `
        ($cpuValues | Measure-Object -Average).Average, `
        ($cpuValues | Measure-Object -Maximum).Maximum)
    if ($goroValues.Count -gt 0) {
        Write-Host ("  Goroutine  start={0}  end={1}  peak={2}" -f `
            $goroValues[0], $goroValues[-1], ($goroValues | Measure-Object -Maximum).Maximum)
    }
}
