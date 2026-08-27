# cwd: repo root. Uses D:\xAI\squad-oc-dummy only (never this git repo).
# Builds squad-oc.exe, inits the dummy, starts or attaches to
# opencode serve on 127.0.0.1:4096, run -p TRACEOK, asserts traces --json/--export.
# Unless -SkipLangfuse: attach or start local Langfuse v4, second run with OTLP, poll traces API.
# Leaves serve running. Does not stop a Langfuse this script did not start.
# Does not change live-e2e.ps1 (PONG check stays put).
param([switch]$SkipLangfuse)

$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $PSScriptRoot
$Dummy = "D:\xAI\squad-oc-dummy"
$BaseURL = "http://127.0.0.1:4096"
$LangfuseURL = "http://127.0.0.1:3000"
$Prompt = "Reply with exactly TRACEOK and nothing else."
$LangfusePrompt = "Reply with exactly LANGFUSEOK and nothing else."
$Log = Join-Path $Dummy "live-traces.log"
$MainDumpDir = "D:\xAI\squad-opencode\.playwright-mcp"
$WorktreeDumpDir = Join-Path $RepoRoot ".playwright-mcp"
$MainDump = Join-Path $MainDumpDir "dummy-traces-summary.json"
$WorktreeDump = Join-Path $WorktreeDumpDir "dummy-traces-summary.json"
$MainLangfuseDump = Join-Path $MainDumpDir "dummy-langfuse-trace.json"
$WorktreeLangfuseDump = Join-Path $WorktreeDumpDir "dummy-langfuse-trace.json"

$Pk = "pk-lf-local-squad-oc"
$Sk = "sk-lf-local-squad-oc"

$startedServe = $false
$startedLangfuse = $false
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

function Clear-OtlpEnv {
    foreach ($n in @(
            "OTEL_EXPORTER_OTLP_ENDPOINT",
            "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
            "OTEL_EXPORTER_OTLP_PROTOCOL",
            "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL",
            "OTEL_EXPORTER_OTLP_HEADERS",
            "OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT"
        )) {
        Remove-Item "Env:$n" -ErrorAction SilentlyContinue
    }
}

function Write-JsonDump {
    param(
        [string[]]$Paths,
        [object]$Record
    )
    $json = $Record | ConvertTo-Json -Depth 12
    foreach ($p in $Paths) {
        $dir = Split-Path -Parent $p
        if (-not (Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir | Out-Null
        }
        Set-Content -Path $p -Value $json -Encoding utf8
        Write-Log "wrote $p"
    }
    return $json
}

function Test-LangfuseHealthy {
    foreach ($url in @("$LangfuseURL/api/public/health", "$LangfuseURL/")) {
        try {
            $r = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 3
            if ($r.StatusCode -ge 200 -and $r.StatusCode -lt 500) {
                return $true
            }
        } catch {
        }
    }
    return $false
}

function Get-LangfuseBasicB64 {
    $pair = "{0}:{1}" -f $Pk, $Sk
    return [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes($pair))
}

function Wait-LangfuseReady {
    param([int]$Seconds = 180)
    $deadline = (Get-Date).AddSeconds($Seconds)
    while ((Get-Date) -lt $deadline) {
        if (Test-LangfuseHealthy) {
            return
        }
        Start-Sleep -Seconds 2
    }
    throw "Langfuse at $LangfuseURL never became ready"
}

function Invoke-LangfuseGet {
    param(
        [string]$Url,
        [hashtable]$Headers
    )
    try {
        $r = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 10 -Headers $Headers
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

# This-run gate: both span names. LANGFUSEOK only if the payload actually has I/O fields.
function Test-LangfuseThisRun {
    param([string]$Body)
    if ([string]::IsNullOrWhiteSpace($Body)) {
        return $false
    }
    if ($Body -notlike "*squad-oc.run*" -or $Body -notlike "*gen_ai.chat*") {
        return $false
    }
    $hasIO = ($Body -like "*gen_ai.input.messages*") -or
        ($Body -like "*gen_ai.output.messages*") -or
        ($Body -match '"input"\s*:') -or
        ($Body -match '"output"\s*:')
    if ($hasIO -and $Body -notlike "*LANGFUSEOK*") {
        return $false
    }
    return $true
}

function Get-LangfuseTraces {
    param(
        [string]$B64,
        [string]$FromStartTime
    )
    $headers = @{ Authorization = "Basic $B64" }
    # Try legacy traces first (v4 events_only → 404). 2xx only counts if this-run markers match.
    $tracesUrl = "$LangfuseURL/api/public/traces?limit=20"
    $tracesGot = Invoke-LangfuseGet -Url $tracesUrl -Headers $headers
    if ($tracesGot.status -ge 200 -and $tracesGot.status -lt 300 -and (Test-LangfuseThisRun $tracesGot.body)) {
        return $tracesGot
    }
    # v4 replacement, windowed so leftover observations cannot match.
    $obsUrl = "$LangfuseURL/api/public/v2/observations?limit=20"
    if (-not [string]::IsNullOrWhiteSpace($FromStartTime)) {
        $obsUrl = $obsUrl + "&fromStartTime=" + [uri]::EscapeDataString($FromStartTime)
    }
    $obsGot = Invoke-LangfuseGet -Url $obsUrl -Headers $headers
    if ($obsGot.status -ge 200 -and $obsGot.status -lt 300 -and (Test-LangfuseThisRun $obsGot.body)) {
        return $obsGot
    }
    if ($null -ne $obsGot.url) {
        return $obsGot
    }
    return $tracesGot
}

try {
    Set-Location $RepoRoot

    if (-not (Test-Path $Dummy)) {
        throw "dummy missing: $Dummy (never use this git clone; never squad-oc-dummy-v2)"
    }
    if (Test-Path $Log) {
        Remove-Item $Log -Force
    }
    Write-Log "repo=$RepoRoot dummy=$Dummy skipLangfuse=$SkipLangfuse"

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

        Clear-OtlpEnv
        $parent = $null
        $child = $null
        $spans = $null
        $runOut = ""
        for ($attempt = 1; $attempt -le 3; $attempt++) {
            $attemptStart = (Get-Date).AddSeconds(-2)
            Write-Log "run -p TRACEOK (no OTLP env) attempt $attempt"
            $runOut = & $exe run -p $Prompt 2>&1 | Out-String
            Write-Host $runOut
            Add-Content -Path $Log -Value $runOut
            if ($LASTEXITCODE -ne 0) {
                Write-Log "run attempt $attempt exited $LASTEXITCODE"
                continue
            }

            Write-Log "traces --json --last 8"
            $tracesJson = & $exe traces --json --last 8 | Out-String
            if ($LASTEXITCODE -ne 0) {
                throw "squad-oc traces --json exited $LASTEXITCODE"
            }
            Write-Host $tracesJson
            Add-Content -Path $Log -Value $tracesJson
            $spans = $tracesJson | ConvertFrom-Json
            if ($null -eq $spans) {
                throw "traces --json produced no spans"
            }
            $child = @(
                $spans | Where-Object {
                    $_.name -eq "gen_ai.chat" -and
                    [string]$_.prompt -like "*TRACEOK*" -and
                    -not [string]::IsNullOrWhiteSpace([string]$_.completion) -and
                    ([datetime]$_.start) -ge $attemptStart
                }
            ) | Select-Object -Last 1
            if ($null -eq $child) {
                Write-Log ("attempt {0}: gen_ai.chat missing completion (OpenCode empty/upstream flake)" -f $attempt)
                continue
            }
            $parent = @(
                $spans | Where-Object { $_.name -eq "squad-oc.run" -and $_.traceId -eq $child.traceId }
            ) | Select-Object -Last 1
            if ($null -ne $parent) {
                break
            }
        }
        if ($null -eq $parent -or $null -eq $child) {
            throw "traces --json missing parent squad-oc.run + child gen_ai.chat with local prompt/completion"
        }
        if ([string]::IsNullOrWhiteSpace([string]$child.model)) {
            throw "gen_ai.chat missing model from OpenCode"
        }
        $inTok = 0
        $outTok = 0
        if ($null -ne $child.inputTokens) { $inTok = [int]$child.inputTokens }
        if ($null -ne $child.outputTokens) { $outTok = [int]$child.outputTokens }
        if ($inTok -eq 0 -and $outTok -eq 0) {
            Write-Log "WARN gen_ai.chat tokens are 0 (OpenCode omitempty / empty Info)"
        }
        Write-Log ("PASS jsonl: parent={0} child={1} model={2} tokens={3}/{4}" -f $parent.name, $child.name, $child.model, $inTok, $outTok)

        $exportPath = Join-Path $env:TEMP ("squad-oc-traces-export-{0}.json" -f [guid]::NewGuid().ToString("n"))
        Write-Log "traces --export $exportPath --last 8"
        Invoke-SquadOc -Exe $exe -CommandArgs @("traces", "--export", $exportPath, "--last", "8")
        $exportText = Get-Content -Raw -Path $exportPath
        if ($exportText -notlike "*gen_ai.*") {
            throw "traces --export missing gen_ai. metadata"
        }
        if ($exportText -like "*gen_ai.input.messages*") {
            throw "traces --export leaked gen_ai.input.messages"
        }
        if ($exportText -like "*$Prompt*") {
            throw "traces --export leaked prompt text"
        }
        Write-Log "PASS export: has gen_ai. metadata, no bodies"

        $record = [ordered]@{
            dummy         = $Dummy
            startedServe  = $startedServe
            skipLangfuse  = [bool]$SkipLangfuse
            prompt        = $Prompt
            runPreview    = $runOut.Substring(0, [Math]::Min(240, $runOut.Length))
            parentName    = $parent.name
            childName     = $child.name
            model         = $child.model
            provider      = $child.provider
            inputTokens   = $child.inputTokens
            outputTokens  = $child.outputTokens
            promptSeen    = $child.prompt
            completion    = $child.completion
            exportPath    = $exportPath
            traces        = $spans
        }
        Write-JsonDump -Paths @($MainDump, $WorktreeDump) -Record $record | Out-Null

        if ($SkipLangfuse) {
            Write-Log "Langfuse skipped: -SkipLangfuse"
        } else {
            if (Test-LangfuseHealthy) {
                Write-Log "attaching to existing Langfuse $LangfuseURL"
            } else {
                $docker = Get-Command docker -ErrorAction SilentlyContinue
                if (-not $docker) {
                    throw "docker not on PATH (needed unless -SkipLangfuse)"
                }
                $composeFile = Join-Path $RepoRoot "scripts\langfuse\docker-compose.yml"
                $envFile = Join-Path $RepoRoot "scripts\langfuse\.env"
                if (-not (Test-Path $composeFile)) {
                    throw "missing $composeFile"
                }
                Write-Log "docker compose -f scripts/langfuse/docker-compose.yml up -d"
                & docker compose --project-name squad-oc-langfuse --env-file $envFile -f $composeFile up -d
                if ($LASTEXITCODE -ne 0) {
                    throw "docker compose up failed ($LASTEXITCODE)"
                }
                $startedLangfuse = $true
                Write-Log "started Langfuse compose (leaving it running); waiting for web"
                Wait-LangfuseReady -Seconds 180
            }

            $b64 = Get-LangfuseBasicB64
            $env:OTEL_EXPORTER_OTLP_ENDPOINT = "http://127.0.0.1:3000/api/public/otel"
            $env:OTEL_EXPORTER_OTLP_PROTOCOL = "http/protobuf"
            $env:OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT = "true"
            $env:OTEL_EXPORTER_OTLP_HEADERS = "Authorization=Basic $b64,x-langfuse-ingestion-version=4"
            Remove-Item Env:OTEL_EXPORTER_OTLP_TRACES_ENDPOINT -ErrorAction SilentlyContinue
            Remove-Item Env:OTEL_EXPORTER_OTLP_TRACES_PROTOCOL -ErrorAction SilentlyContinue

            $fromStart = (Get-Date).ToUniversalTime().AddSeconds(-2).ToString("yyyy-MM-ddTHH:mm:ssZ")
            Write-Log "Langfuse poll window fromStartTime=$fromStart"

            $lfRunOut = ""
            $lfOk = $false
            for ($attempt = 1; $attempt -le 3; $attempt++) {
                Write-Log "run -p LANGFUSEOK (OTLP http/protobuf + capture on) attempt $attempt"
                $lfRunOut = & $exe run -p $LangfusePrompt 2>&1 | Out-String
                Write-Host $lfRunOut
                Add-Content -Path $Log -Value $lfRunOut
                if ($LASTEXITCODE -eq 0) {
                    $lfOk = $true
                    break
                }
                Write-Log "langfuse run attempt $attempt exited $LASTEXITCODE"
            }
            if (-not $lfOk) {
                throw "squad-oc run (langfuse) failed after retries"
            }

            $got = $null
            $deadline = (Get-Date).AddSeconds(60)
            while ((Get-Date) -lt $deadline) {
                $got = Get-LangfuseTraces -B64 $b64 -FromStartTime $fromStart
                Write-Log ("GET {0} status={1}" -f $got.url, $got.status)
                if ($got.status -ge 200 -and $got.status -lt 300 -and (Test-LangfuseThisRun $got.body)) {
                    Write-Log "PASS Langfuse API found this-run parent+child"
                    break
                }
                $got = $null
                Start-Sleep -Seconds 2
            }
            if ($null -eq $got -or -not (Test-LangfuseThisRun $got.body)) {
                throw "Langfuse GET never showed this-run squad-oc.run + gen_ai.chat"
            }

            $lfRecord = [ordered]@{
                dummy           = $Dummy
                startedLangfuse = $startedLangfuse
                endpoint        = $env:OTEL_EXPORTER_OTLP_ENDPOINT
                protocol        = $env:OTEL_EXPORTER_OTLP_PROTOCOL
                fromStartTime   = $fromStart
                prompt          = $LangfusePrompt
                runPreview      = $lfRunOut.Substring(0, [Math]::Min(240, $lfRunOut.Length))
                tracesStatus    = $got.status
                tracesUrl       = $got.url
                traces          = $got.body
            }
            Write-JsonDump -Paths @($MainLangfuseDump, $WorktreeLangfuseDump) -Record $lfRecord | Out-Null
        }
    } finally {
        Set-Location $prev
        Clear-OtlpEnv
    }

    if ($startedServe) {
        Write-Log "leaving serve running (pid=$($serveProc.Id))"
    } else {
        Write-Log "left existing serve running"
    }
    if (-not $SkipLangfuse) {
        if ($startedLangfuse) {
            Write-Log "leaving Langfuse compose running (this script started it)"
        } else {
            Write-Log "left existing Langfuse running"
        }
    }
    Write-Log "live-traces ok"
    exit 0
} catch {
    Write-Log ("FAIL: " + $_.Exception.Message)
    throw
}
