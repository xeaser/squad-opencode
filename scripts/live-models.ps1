# cwd: repo root. Uses D:\xAI\squad-oc-dummy only (never this git repo).
# Builds squad-oc.exe, pins per-agent models, starts or attaches to
# opencode serve on 127.0.0.1:4096, asserts host files + GET /config.
# Leaves serve running. Does not restore dummy model pins.
# Does not change live-e2e.ps1 (PONG check stays put).

$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $PSScriptRoot
$Dummy = "D:\xAI\squad-oc-dummy"
$BaseURL = "http://127.0.0.1:4096"
$Log = Join-Path $Dummy "live-models.log"
$MainDumpDir = "D:\xAI\squad-opencode\.playwright-mcp"
$MainDump = Join-Path $MainDumpDir "dummy-per-agent-model-config.json"
$WorktreeDumpDir = Join-Path $RepoRoot ".playwright-mcp"
$WorktreeDump = Join-Path $WorktreeDumpDir "dummy-per-agent-model-config.json"

$SquadModel = "opencode/big-pickle"
$LeadModel = "opencode/hy3-free"

$startedServe = $false
$serveProc = $null

function Write-Log {
    param([string]$Message)
    $line = "[{0}] {1}" -f (Get-Date -Format "yyyy-MM-ddTHH:mm:ssK"), $Message
    Write-Host $line
    $parent = Split-Path -Parent $Log
    if (Test-Path $parent) {
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

function Get-OpencodeExe {
    $cmd = Get-Command opencode -ErrorAction SilentlyContinue
    if (-not $cmd) {
        throw "opencode not on PATH"
    }
    $src = $cmd.Source
    if ($src -like "*.ps1") {
        $exe = Join-Path (Split-Path $src) "node_modules\opencode-ai\bin\opencode.exe"
        if (Test-Path $exe) {
            return $exe
        }
    }
    return $src
}

function Resolve-HostAgentFile {
    param([string]$RoleId)
    $agents = Join-Path $Dummy ".opencode\agents"
    $candidates = @(Join-Path $agents "$RoleId.md")
    $office = @{
        lead     = "michael"
        frontend = "jim"
        backend  = "dwight"
        tester   = "pam"
    }
    if ($office.ContainsKey($RoleId)) {
        $candidates += (Join-Path $agents ($office[$RoleId] + ".md"))
    }
    foreach ($p in $candidates) {
        if (Test-Path $p) {
            return $p
        }
    }
    throw "host agent file not found for $RoleId (tried: $($candidates -join ', '))"
}

function Assert-FileContains {
    param(
        [string]$Path,
        [string]$Needle,
        [string]$Label
    )
    $text = Get-Content -Raw -Path $Path
    if ($text -notlike "*$Needle*") {
        throw "$Label missing '$Needle' in $Path"
    }
    Write-Log ("PASS file: {0} has '{1}'" -f $Label, $Needle)
}

function Invoke-ServeGet {
    param([string]$Url)
    try {
        $r = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 10
        return [ordered]@{
            url    = $Url
            status = [int]$r.StatusCode
            body   = $r.Content
        }
    } catch {
        $status = 0
        $resp = $_.Exception.Response
        if ($null -ne $resp -and $null -ne $resp.StatusCode) {
            $status = [int]$resp.StatusCode
        }
        return [ordered]@{
            url    = $Url
            status = $status
            body   = $null
            error  = $_.Exception.Message
        }
    }
}

try {
    Set-Location $RepoRoot

    if (-not (Test-Path $Dummy)) {
        throw "dummy missing: $Dummy (never use this git clone; never squad-oc-dummy-v2)"
    }
    if (Test-Path $Log) {
        Remove-Item $Log -Force
    }
    Write-Log "repo=$RepoRoot dummy=$Dummy"

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
            Invoke-SquadOc -Exe $exe -CommandArgs @("init", "--preset", "default", "--description", "dummy")
        } else {
            Write-Log "already initialized"
        }

        Invoke-SquadOc -Exe $exe -CommandArgs @("recast")
        Invoke-SquadOc -Exe $exe -CommandArgs @("cast", "--model", "squad", $SquadModel)
        Invoke-SquadOc -Exe $exe -CommandArgs @("cast", "--model", "Lead", $LeadModel)
        Invoke-SquadOc -Exe $exe -CommandArgs @("cast", "--model", "Tester", "-")
        Invoke-SquadOc -Exe $exe -CommandArgs @("status")

        $squadFile = Join-Path $Dummy ".opencode\agents\squad.md"
        if (-not (Test-Path $squadFile)) {
            throw "missing $squadFile"
        }
        Assert-FileContains -Path $squadFile -Needle "model: $SquadModel" -Label "squad.md"
        Assert-FileContains -Path $squadFile -Needle "You are **Squad**" -Label "squad.md"

        $leadFile = Resolve-HostAgentFile -RoleId "lead"
        Write-Log "Lead host file: $leadFile"
        Assert-FileContains -Path $leadFile -Needle "model: $LeadModel" -Label (Split-Path $leadFile -Leaf)

        $testerFile = Resolve-HostAgentFile -RoleId "tester"
        Write-Log "Tester host file: $testerFile"
        Assert-FileContains -Path $testerFile -Needle "model: $SquadModel" -Label (Split-Path $testerFile -Leaf)

        $opencodeExe = Get-OpencodeExe
        Write-Log "opencode=$opencodeExe"

        if (Assert-Port4096Safe) {
            Write-Log "attaching to existing localhost:4096"
        } else {
            Write-Log "starting opencode serve --hostname 127.0.0.1 --port 4096"
            $serveProc = Start-Process -FilePath $opencodeExe -ArgumentList @("serve", "--hostname", "127.0.0.1", "--port", "4096") -WorkingDirectory $Dummy -PassThru -WindowStyle Hidden
            $startedServe = $true
            Write-Log "started serve pid=$($serveProc.Id) (leaving it running)"
        }
        Wait-ServeReady

        $health = Invoke-ServeGet "$BaseURL/global/health"
        Write-Log ("GET /global/health status={0}" -f $health.status)
        if ($health.status -lt 200 -or $health.status -ge 500) {
            throw "GET /global/health failed: $($health.error)"
        }

        $dirBack = [uri]::EscapeDataString($Dummy)
        $dirFwd = [uri]::EscapeDataString(($Dummy -replace '\\', '/'))
        $configUrls = @(
            "$BaseURL/config?directory=$dirBack",
            "$BaseURL/config?directory=$dirFwd",
            "$BaseURL/config",
            "$BaseURL/project",
            "$BaseURL/agent?directory=$dirBack",
            "$BaseURL/agent?directory=$dirFwd",
            "$BaseURL/agent"
        )

        $probes = @()
        $chosen = $null
        foreach ($url in $configUrls) {
            $got = Invoke-ServeGet $url
            $probes += $got
            $preview = ""
            if ($got.body) {
                $preview = $got.body.Substring(0, [Math]::Min(80, $got.body.Length))
            }
            Write-Log ("GET {0} status={1} preview={2}" -f $url, $got.status, $preview)
            if ($null -eq $chosen -and $got.status -ge 200 -and $got.status -lt 300 -and $got.body) {
                $chosen = $got
            }
        }

        $record = [ordered]@{
            dummy        = $Dummy
            startedServe = $startedServe
            health       = $health
            chosenUrl    = if ($chosen) { $chosen.url } else { $null }
            probes       = $probes
        }
        $json = $record | ConvertTo-Json -Depth 12
        foreach ($dir in @($MainDumpDir, $WorktreeDumpDir)) {
            if (-not (Test-Path $dir)) {
                New-Item -ItemType Directory -Path $dir | Out-Null
            }
        }
        Set-Content -Path $MainDump -Value $json -Encoding utf8
        Set-Content -Path $WorktreeDump -Value $json -Encoding utf8
        Write-Log "wrote $MainDump"
        Write-Log "wrote $WorktreeDump"

        $haystack = $json
        if ($haystack -notlike "*$SquadModel*") {
            throw "serve JSON missing '$SquadModel' (see $MainDump)"
        }
        if ($haystack -notlike "*$LeadModel*") {
            throw "serve JSON missing '$LeadModel' (see $MainDump)"
        }
        Write-Log "PASS serve API: JSON contains $SquadModel and $LeadModel"

        Write-Log "TUI/Playwright skipped (serve JSON is the required artifact)"
    } finally {
        Set-Location $prev
    }

    if ($startedServe) {
        Write-Log "leaving serve running (pid=$($serveProc.Id))"
    } else {
        Write-Log "left existing serve running"
    }
    Write-Log "live-models ok"
    exit 0
} catch {
    Write-Log ("FAIL: " + $_.Exception.Message)
    throw
}
