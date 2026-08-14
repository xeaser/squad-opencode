# cwd: repo root. Uses D:\xAI\squad-oc-dummy only (never this git repo).
# Builds squad-oc.exe, inits the dummy, starts or attaches to
# opencode serve on 127.0.0.1:4096, then doctor + run (+ optional recast).
param([switch]$SkipTui)

$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $PSScriptRoot
$Dummy = "D:\xAI\squad-oc-dummy"
$BaseURL = "http://127.0.0.1:4096"
$Prompt = "Reply with exactly PONG and nothing else."
$Log = Join-Path $Dummy "live-e2e.log"

$startedServe = $false
$serveProc = $null

function Write-Log {
    param([string]$Message)
    $line = "[{0}] {1}" -f (Get-Date -Format "yyyy-MM-ddTHH:mm:ssK"), $Message
    Write-Host $line
    if (Test-Path (Split-Path -Parent $Log)) {
        Add-Content -Path $Log -Value $line
    }
}

function Get-Port4096Listeners {
    Get-NetTCPConnection -LocalPort 4096 -State Listen -ErrorAction SilentlyContinue
}

function Assert-Port4096Safe {
    $listeners = @(Get-Port4096Listeners)
    if ($listeners.Count -eq 0) {
        return $false
    }
    $ok = @("127.0.0.1", "::1", "localhost")
    $bad = @($listeners | Where-Object { $ok -notcontains $_.LocalAddress })
    if ($bad.Count -gt 0) {
        $addrs = ($bad | ForEach-Object { $_.LocalAddress } | Sort-Object -Unique) -join ", "
        throw "port 4096 is bound on non-localhost ($addrs); refusing to steal another server"
    }
    return $true
}

function Wait-ServeReady {
    param([int]$Seconds = 20)
    $deadline = (Get-Date).AddSeconds($Seconds)
    while ((Get-Date) -lt $deadline) {
        try {
            $r = Invoke-WebRequest -Uri "$BaseURL/global/health" -UseBasicParsing -TimeoutSec 2
            if ($r.StatusCode -ge 200 -and $r.StatusCode -lt 500) {
                return
            }
        } catch {
            Start-Sleep -Milliseconds 250
        }
    }
    throw "opencode serve at $BaseURL never became ready"
}

function Invoke-SquadOc {
    param(
        [string]$Exe,
        [string[]]$CommandArgs
    )
    Write-Log ("squad-oc " + ($CommandArgs -join " "))
    & $Exe @CommandArgs
    if ($LASTEXITCODE -ne 0) {
        throw "squad-oc $($CommandArgs[0]) exited $LASTEXITCODE"
    }
}

try {
    Set-Location $RepoRoot

    if (-not (Test-Path $Dummy)) {
        New-Item -ItemType Directory -Path $Dummy | Out-Null
    }
    if (Test-Path $Log) {
        Remove-Item $Log -Force
    }
    Write-Log "repo=$RepoRoot dummy=$Dummy"

    if (-not (Test-Path (Join-Path $Dummy ".git"))) {
        Write-Log "git init $Dummy"
        git init $Dummy | Out-Host
        if ($LASTEXITCODE -ne 0) {
            throw "git init failed in $Dummy"
        }
    }

    Write-Log "go build -o squad-oc.exe ./cmd/squad-oc"
    go build -o squad-oc.exe ./cmd/squad-oc
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed"
    }
    $exe = Join-Path $RepoRoot "squad-oc.exe"
    if (-not (Test-Path $exe)) {
        throw "squad-oc.exe missing after build"
    }

    $prev = Get-Location
    Set-Location $Dummy
    try {
        if (-not (Test-Path (Join-Path $Dummy ".squad\config.json"))) {
            Invoke-SquadOc -Exe $exe -CommandArgs @("init", "--preset", "default")
        } else {
            Write-Log "already initialized"
        }

        $opencode = Get-Command opencode -ErrorAction SilentlyContinue
        if (-not $opencode) {
            throw "opencode not on PATH"
        }

        if (Assert-Port4096Safe) {
            Write-Log "attaching to existing localhost:4096"
        } else {
            Write-Log "starting opencode serve --hostname 127.0.0.1 --port 4096"
            $serveProc = Start-Process -FilePath $opencode.Source -ArgumentList @("serve", "--hostname", "127.0.0.1", "--port", "4096") -WorkingDirectory $Dummy -PassThru -WindowStyle Hidden
            $startedServe = $true
            Write-Log "started serve pid=$($serveProc.Id)"
        }
        Wait-ServeReady

        Invoke-SquadOc -Exe $exe -CommandArgs @("doctor")

        Write-Log "run -p PONG"
        $runOut = & $exe run -p $Prompt 2>&1 | Out-String
        Write-Host $runOut
        Add-Content -Path $Log -Value $runOut
        if ($LASTEXITCODE -ne 0) {
            throw "squad-oc run exited $LASTEXITCODE"
        }
        if ($runOut -notmatch "PONG" -and [string]::IsNullOrWhiteSpace($runOut)) {
            throw "run produced no assistant text"
        }
        if ($runOut -notmatch "PONG") {
            Write-Log "run paraphrased (no exact PONG); accepted non-empty reply"
        } else {
            Write-Log "run ok (PONG)"
        }

        $probeAgent = Join-Path $Dummy ".opencode\agents\liveprobe.md"
        if (Test-Path $probeAgent) {
            Write-Log "recast smoke: liveprobe.md already present"
        } else {
            Invoke-SquadOc -Exe $exe -CommandArgs @("cast", "--add", "LiveProbe", "--role", "Probe")
            if (-not (Test-Path $probeAgent)) {
                throw "expected .opencode/agents/liveprobe.md after cast --add LiveProbe"
            }
            Write-Log "recast smoke: wrote liveprobe.md"
        }

        if ($SkipTui) {
            Write-Log "TUI skipped: -SkipTui"
        }
    } finally {
        Set-Location $prev
    }

    Write-Log "live-e2e ok"
    exit 0
} catch {
    Write-Log ("FAIL: " + $_.Exception.Message)
    throw
} finally {
    if ($startedServe -and $null -ne $serveProc) {
        Write-Log "stopping serve pid=$($serveProc.Id)"
        try {
            if (-not $serveProc.HasExited) {
                Stop-Process -Id $serveProc.Id -Force -ErrorAction SilentlyContinue
                # children of opencode serve (node)
                Get-CimInstance Win32_Process -Filter "ParentProcessId=$($serveProc.Id)" -ErrorAction SilentlyContinue |
                    ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
            }
        } catch {
            Write-Log ("stop serve: " + $_.Exception.Message)
        }
    }
}
