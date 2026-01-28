<#
.SYNOPSIS
    Windows test script for the adde.exe CLI.
.DESCRIPTION
    Runs test cases against adde.exe. Use from the go/ directory after building:
    cd go
    go build -o adde.exe .\cmd\adde
    .\test_adde_cli.ps1
    Set $env:ADDE_EXE to the full path of adde.exe to use a specific binary.
.EXAMPLE
    .\test_adde_cli.ps1
    .\test_adde_cli.ps1 -SkipE2E
.EXAMPLE
    .\test_adde_cli.ps1 -BusyBox
    Run E2E using busybox and a .sh script (echo 42) instead of python:3.11-slim.
#>

param(
    [switch]$SkipE2E,
    [switch]$BusyBox
)

$ErrorActionPreference = "Stop"
$Failed = 0
$Passed = 0

function Find-AddeExe {
    if ($env:ADDE_EXE -and (Test-Path $env:ADDE_EXE)) {
        return $env:ADDE_EXE
    }
    $candidates = @(
        (Join-Path $PSScriptRoot "adde.exe"),
        (Join-Path $PSScriptRoot ".\adde.exe")
    )
    foreach ($c in $candidates) {
        $resolved = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($c)
        if ($resolved -and (Test-Path $resolved)) {
            return $resolved
        }
    }
    throw "adde.exe not found. Build with: go build -o adde.exe .\cmd\adde"
}

function Run-Adde {
    param([string]$Tool, [string]$Payload)
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $AddeExe
    if ($Tool -eq "") {
        $psi.Arguments = ""
    } else {
        # Quote JSON so the exe receives it as one argument
        $psi.Arguments = if ($Payload -eq "") { $Tool } else { "$Tool `"$($Payload -replace '"','\"')`"" }
    }
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true
    $psi.UseShellExecute = $false
    $psi.CreateNoWindow = $true
    $p = [System.Diagnostics.Process]::Start($psi)
    $stdout = $p.StandardOutput.ReadToEnd()
    $stderr = $p.StandardError.ReadToEnd()
    $p.WaitForExit(30000) | Out-Null
    return @{ ExitCode = $p.ExitCode; Stdout = $stdout; Stderr = $stderr }
}

function Assert-Equal {
    param($Expected, $Actual, [string]$Name)
    if ($Expected -ne $Actual) {
        Write-Host "FAIL $Name : expected $Expected, got $Actual" -ForegroundColor Red
        $script:Failed++
        return $false
    }
    Write-Host "PASS $Name" -ForegroundColor Green
    $script:Passed++
    return $true
}

function Assert-Contains {
    param([string]$Haystack, [string]$Needle, [string]$Name)
    if ($Haystack -notmatch [regex]::Escape($Needle)) {
        Write-Host "FAIL $Name : expected output to contain '$Needle'" -ForegroundColor Red
        $script:Failed++
        return $false
    }
    Write-Host "PASS $Name" -ForegroundColor Green
    $script:Passed++
    return $true
}

# Resolve exe once
$AddeExe = Find-AddeExe
Write-Host "Using: $AddeExe" -ForegroundColor Cyan
Write-Host ""

# --- Test: no args -> usage, exit 2 ---
$r = Run-Adde -Tool "" -Payload ""
Assert-Equal -Expected 2 -Actual $r.ExitCode -Name "usage: exit code 2"
Assert-Contains -Haystack ($r.Stderr + $r.Stdout) -Needle "usage" -Name "usage: message"

# --- Test: unknown tool -> exit 1 ---
$r = Run-Adde -Tool "no_such_tool" -Payload "{}"
Assert-Equal -Expected 1 -Actual $r.ExitCode -Name "unknown tool: exit 1"
Assert-Contains -Haystack $r.Stderr -Needle "unknown" -Name "unknown tool: message"

# --- Test: create_runtime_env invalid JSON -> exit 1 ---
$r = Run-Adde -Tool "create_runtime_env" -Payload "{invalid}"
Assert-Equal -Expected 1 -Actual $r.ExitCode -Name "create_runtime_env invalid JSON: exit 1"

# --- Test: execute_code_block invalid JSON -> exit 1 ---
$r = Run-Adde -Tool "execute_code_block" -Payload "not json"
Assert-Equal -Expected 1 -Actual $r.ExitCode -Name "execute_code_block invalid JSON: exit 1"

# --- Test: get_container_logs invalid JSON -> exit 1 ---
$r = Run-Adde -Tool "get_container_logs" -Payload "[]"
Assert-Equal -Expected 1 -Actual $r.ExitCode -Name "get_container_logs invalid JSON: exit 1"

# --- Test: cleanup_env invalid JSON -> exit 1 ---
$r = Run-Adde -Tool "cleanup_env" -Payload "{bad}"
Assert-Equal -Expected 1 -Actual $r.ExitCode -Name "cleanup_env invalid JSON: exit 1"

# --- Test: pull_image invalid JSON -> exit 1 ---
$r = Run-Adde -Tool "pull_image" -Payload "[]"
Assert-Equal -Expected 1 -Actual $r.ExitCode -Name "pull_image invalid JSON: exit 1"

# --- E2E (optional, requires Docker) ---
if (-not $SkipE2E) {
    Write-Host ""
    $image = if ($BusyBox) { "busybox" } else { "python:3.11-slim" }
    # Pull image so create_runtime_env works even when image is not already present
    $pullPayload = "{`"image`":`"$image`"}"
    $r = Run-Adde -Tool "pull_image" -Payload $pullPayload
    if ($r.ExitCode -ne 0) {
        Write-Host "SKIP E2E (pull_image failed): $($r.Stderr)" -ForegroundColor Yellow
    } else {
    $filename = if ($BusyBox) { "t.sh" } else { "t.py" }
    $codeContent = if ($BusyBox) { "echo 42" } else { "print(42)" }
    Write-Host "E2E: create_runtime_env ($image)..." -ForegroundColor Cyan
    $createPayload = "{`"image`":`"$image`",`"dependencies`":[],`"env_vars`":{},`"network`":false}"
    $r = Run-Adde -Tool "create_runtime_env" -Payload $createPayload
    if ($r.ExitCode -ne 0) {
        Write-Host "SKIP E2E (Docker may be down): create_runtime_env failed" -ForegroundColor Yellow
    } else {
        $createOut = $r.Stdout.Trim() | ConvertFrom-Json
        if ($createOut.error) {
            Write-Host "SKIP E2E: $($createOut.error)" -ForegroundColor Yellow
        } else {
            $cid = $createOut.container_id
            try {
                $execPayload = "{`"container_id`":`"$cid`",`"filename`":`"$filename`",`"code_content`":`"$($codeContent -replace '"','\"')`",`"timeout_sec`":15}"
                $r = Run-Adde -Tool "execute_code_block" -Payload $execPayload
                Assert-Equal -Expected 0 -Actual $r.ExitCode -Name "e2e execute_code_block"
                $execOut = $r.Stdout.Trim() | ConvertFrom-Json
                if ($execOut.log -and $execOut.log.stdout -match "42") {
                    Write-Host "PASS e2e execute_code_block stdout" -ForegroundColor Green
                    $script:Passed++
                } else {
                    Write-Host "FAIL e2e execute_code_block stdout" -ForegroundColor Red
                    $script:Failed++
                }
                $logsPayload = "{`"container_id`":`"$cid`",`"tail_lines`":10}"
                $r = Run-Adde -Tool "get_container_logs" -Payload $logsPayload
                Assert-Equal -Expected 0 -Actual $r.ExitCode -Name "e2e get_container_logs"
            } finally {
                $cleanPayload = "{`"container_id`":`"$cid`"}"
                Run-Adde -Tool "cleanup_env" -Payload $cleanPayload | Out-Null
            }
        }
    }
    }
}

# --- Summary ---
Write-Host ""
Write-Host "---" -ForegroundColor Cyan
Write-Host "Passed: $Passed  Failed: $Failed" -ForegroundColor $(if ($Failed -gt 0) { "Red" } else { "Green" })
if ($Failed -gt 0) {
    exit 1
}
