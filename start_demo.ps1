<#
.SYNOPSIS
    TraumaTeam Single-Command Local Launch & Deterministic Smoke-Check Script (Windows).

.DESCRIPTION
    Builds the Rust RFM cdylib, builds the Go weather_ingest gateway with FFI linking,
    starts the Go HTTP/WebSocket/gRPC gateway, verifies /healthz, /api/v1/conditions,
    /api/v1/kinematics, and /ws (PING/PONG), and either:
      - With -CheckOnly: builds the Svelte frontend, completes all smoke checks, and stops the gateway.
      - Without -CheckOnly: launches the Svelte Vite dev server on http://127.0.0.1:5173.

.PARAMETER CheckOnly
    Build frontend (npm run build), run gateway smoke checks, and cleanly stop the gateway.
#>
[CmdletBinding()]
param(
    [switch]$CheckOnly,
    [int]$HttpPort = 8080,
    [int]$GrpcPort = 50051,
    [int]$UiPort = 5173
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$RustDir = Join-Path $RepoRoot "crates\traumateam_rfm"
$RustReleaseDir = Join-Path $RustDir "target\release"
$RustDllPath = Join-Path $RustReleaseDir "traumateam_rfm.dll"
$GoDir = Join-Path $RepoRoot "services\weather_ingest"
$GoBinaryPath = Join-Path $GoDir "weather_ingest.exe"
$FrontendDir = Join-Path $RepoRoot "frontend"

function Write-Step {
    param([string]$Message)
    Write-Host "[+] $Message" -ForegroundColor Cyan
}

function Write-Pass {
    param([string]$Message)
    Write-Host "[PASS] $Message" -ForegroundColor Green
}

function Assert-PortFree {
    param([int]$Port)
    $client = New-Object System.Net.Sockets.TcpClient
    try {
        $async = $client.BeginConnect("127.0.0.1", $Port, $null, $null)
        if ($async.AsyncWaitHandle.WaitOne(200)) {
            try {
                $client.EndConnect($async)
                throw "Port 127.0.0.1:$Port is already in use. Stop the existing process before running start_demo.ps1."
            } catch [System.Net.Sockets.SocketException] {
                # Connection refused => port is free
            }
        }
    } finally {
        $client.Dispose()
    }
}

function Invoke-WebSocketPingCheck {
    param([string]$WsUri)

    $ws = New-Object System.Net.WebSockets.ClientWebSocket
    $cts = New-Object System.Threading.CancellationTokenSource
    $cts.CancelAfter(5000)

    try {
        $uri = [Uri]$WsUri
        $ws.ConnectAsync($uri, $cts.Token).GetAwaiter().GetResult()

        $reqId = "smoke-ping-" + [Guid]::NewGuid().ToString("N").Substring(0, 8)
        $epochMs = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
        $payloadJson = (@{
            action     = "PING"
            request_id = $reqId
            timestamp  = $epochMs
        } | ConvertTo-Json -Compress)

        $sendBytes = [System.Text.Encoding]::UTF8.GetBytes($payloadJson)
        $sendSegment = New-Object 'System.ArraySegment[byte]' -ArgumentList @(,$sendBytes)
        $ws.SendAsync(
            $sendSegment,
            [System.Net.WebSockets.WebSocketMessageType]::Text,
            $true,
            $cts.Token
        ).GetAwaiter().GetResult()

        $ms = New-Object System.IO.MemoryStream
        $recvBuf = New-Object byte[] 4096
        try {
            do {
                $recvSegment = New-Object 'System.ArraySegment[byte]' -ArgumentList @(,$recvBuf)
                $result = $ws.ReceiveAsync($recvSegment, $cts.Token).GetAwaiter().GetResult()
                if ($result.MessageType -eq [System.Net.WebSockets.WebSocketMessageType]::Close) {
                    throw "WebSocket closed unexpectedly before PONG response."
                }
                if ($result.Count -gt 0) {
                    $ms.Write($recvBuf, 0, $result.Count)
                }
            } while (-not $result.EndOfMessage)

            $responseText = [System.Text.Encoding]::UTF8.GetString($ms.ToArray())
        } finally {
            $ms.Dispose()
        }

        $msg = $responseText | ConvertFrom-Json
        if ($msg.event -ne "PONG" -or $msg.request_id -ne $reqId) {
            throw "Unexpected WebSocket PING response: $responseText"
        }

        try {
            $closeCts = New-Object System.Threading.CancellationTokenSource
            $closeCts.CancelAfter(1000)
            $ws.CloseAsync(
                [System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure,
                "smoke check complete",
                $closeCts.Token
            ).GetAwaiter().GetResult()
            $closeCts.Dispose()
        } catch {
            # Ignore benign close timeout
        }

        return $msg
    } finally {
        $cts.Dispose()
        $ws.Dispose()
    }
}

# Ensure standard cargo location is discoverable
$CargoBin = Join-Path $env:USERPROFILE ".cargo\bin"
if ((Test-Path $CargoBin) -and -not ($env:PATH -split ";" | Where-Object { $_ -eq $CargoBin })) {
    $env:PATH = "$CargoBin;$env:PATH"
}

foreach ($tool in @("cargo", "go", "npm")) {
    if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
        throw "Required toolchain command '$tool' was not found on PATH."
    }
}

Write-Host "================================================================================" -ForegroundColor DarkCyan
Write-Host " TRAUMATEAM 4-TIER DEMO LAUNCHER & VERIFICATION HARNESS (WINDOWS)" -ForegroundColor White
Write-Host "================================================================================" -ForegroundColor DarkCyan

# 1. Build Rust shared library (cdylib)
Write-Step "Building Rust RFM physics core (cargo build --release)..."
Push-Location $RustDir
try {
    & cargo build --release
    if ($LASTEXITCODE -ne 0) {
        throw "cargo build --release failed with exit code $LASTEXITCODE."
    }
} finally {
    Pop-Location
}

if (-not (Test-Path $RustDllPath)) {
    throw "Expected Rust shared library not found at: $RustDllPath"
}
Write-Pass "Rust cdylib ready: $RustDllPath"

# 2. Configure FFI dynamic library environment variables
$env:TRAUMATEAM_RFM_LIB_PATH = $RustDllPath
if (-not ($env:PATH -split ";" | Where-Object { $_ -eq $RustReleaseDir })) {
    $env:PATH = "$RustReleaseDir;$env:PATH"
}
Write-Step "Configured TRAUMATEAM_RFM_LIB_PATH=$env:TRAUMATEAM_RFM_LIB_PATH"

# 3. Build Go weather_ingest gateway
Write-Step "Building Go telemetry & compliance gateway (weather_ingest.exe)..."
Push-Location $GoDir
try {
    & go build -o weather_ingest.exe main.go
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed with exit code $LASTEXITCODE."
    }
} finally {
    Pop-Location
}
Write-Pass "Go gateway built: $GoBinaryPath"

# 4. In -CheckOnly mode, build frontend before gateway smoke tests
if ($CheckOnly) {
    Write-Step "Building Svelte 5 frontend production bundle (npm run build)..."
    Push-Location $FrontendDir
    try {
        if (-not (Test-Path (Join-Path $FrontendDir "node_modules"))) {
            Write-Step "Installing frontend dependencies (npm ci)..."
            & npm ci
            if ($LASTEXITCODE -ne 0) {
                throw "npm ci failed with exit code $LASTEXITCODE."
            }
        }
        & npm run build
        if ($LASTEXITCODE -ne 0) {
            throw "npm run build failed with exit code $LASTEXITCODE."
        }
    } finally {
        Pop-Location
    }
    Write-Pass "Frontend production bundle built cleanly."
}

# 5. Start Go gateway and verify healthz + smoke checks
Assert-PortFree -Port $HttpPort
Assert-PortFree -Port $GrpcPort

$LogPrefix = Join-Path ([System.IO.Path]::GetTempPath()) ("traumateam_gateway_" + [Guid]::NewGuid().ToString("N").Substring(0, 8))
$StdoutLog = "$LogPrefix.stdout.log"
$StderrLog = "$LogPrefix.stderr.log"
$GatewayProc = $null

try {
    Write-Step "Starting Go gateway on 127.0.0.1:$HttpPort (HTTP/WS) and 127.0.0.1:$GrpcPort (gRPC)..."
    $GatewayProc = Start-Process `
        -FilePath $GoBinaryPath `
        -ArgumentList @("-http-port", "$HttpPort", "-grpc-port", "$GrpcPort") `
        -WorkingDirectory $GoDir `
        -RedirectStandardOutput $StdoutLog `
        -RedirectStandardError $StderrLog `
        -NoNewWindow `
        -PassThru

    $BaseHttpUrl = "http://127.0.0.1:$HttpPort"
    $HealthUrl = "$BaseHttpUrl/healthz"
    $Deadline = (Get-Date).AddSeconds(20)
    $IsHealthy = $false

    while ((Get-Date) -lt $Deadline) {
        if ($GatewayProc.HasExited) {
            $errTail = if (Test-Path $StderrLog) { (Get-Content $StderrLog -Raw) } else { "" }
            throw "Go gateway exited prematurely with code $($GatewayProc.ExitCode). Logs:`n$errTail"
        }

        try {
            $healthResp = Invoke-RestMethod -Uri $HealthUrl -Method Get -TimeoutSec 2 -UseBasicParsing
            if ($healthResp.status -eq "UP") {
                $IsHealthy = $true
                break
            }
        } catch {
            Start-Sleep -Milliseconds 250
        }
    }

    if (-not $IsHealthy) {
        throw "Timed out waiting for $HealthUrl to report status=UP."
    }
    Write-Pass "Healthcheck UP at $HealthUrl (service=$($healthResp.service))"

    # Smoke Check 1: /api/v1/conditions?station=KLXV
    $CondUrl = "$BaseHttpUrl/api/v1/conditions?station=KLXV"
    $condResp = Invoke-RestMethod -Uri $CondUrl -Method Get -TimeoutSec 5 -UseBasicParsing
    if ($condResp.station_id -ne "KLXV" -or -not $condResp.metar -or $condResp.metar.station_id -ne "KLXV") {
        throw "Unexpected response from $CondUrl"
    }
    Write-Pass "REST /api/v1/conditions?station=KLXV OK (Temp=$($condResp.metar.temperature_c)C, QNH=$($condResp.metar.altimeter_inhg) inHg)"

    # Smoke Check 2: /api/v1/kinematics?station=KLXV
    $KinUrl = "$BaseHttpUrl/api/v1/kinematics?station=KLXV"
    $kinResp = Invoke-RestMethod -Uri $KinUrl -Method Get -TimeoutSec 5 -UseBasicParsing
    if (-not $kinResp.airframe -or $kinResp.verdict -notin @("GO", "CAUTION", "NO-GO")) {
        throw "Unexpected kinematics payload from $KinUrl"
    }
    if ($kinResp.pressure_altitude_ft -le 0 -or $kinResp.max_oge_hover_weight_kg -le 0 -or $kinResp.mass_balance.gross_weight_kg -le 0) {
        throw "Invalid physics telemetry values returned from $KinUrl"
    }
    Write-Pass ("REST /api/v1/kinematics?station=KLXV OK (Airframe=$($kinResp.airframe), Verdict=$($kinResp.verdict), " +
        "PA=$([math]::Round($kinResp.pressure_altitude_ft, 0)) ft, DA=$([math]::Round($kinResp.density_altitude_ft, 0)) ft, " +
        "OGE Margin=$([math]::Round($kinResp.oge_margin_kg, 1)) kg)")

    # Smoke Check 3: WebSocket Handshake + PING/PONG
    $WsUrl = "ws://127.0.0.1:$HttpPort/ws"
    $pongMsg = Invoke-WebSocketPingCheck -WsUri $WsUrl
    Write-Pass "WebSocket $WsUrl PING/PONG OK (seq=$($pongMsg.seq), request_id=$($pongMsg.request_id))"

    if ($CheckOnly) {
        Write-Host ""
        Write-Pass "ALL CHECK-ONLY SMOKE CHECKS PASSED CLEANLY."
        exit 0
    }

    # Interactive mode: print service URLs and launch Svelte Vite dev server
    Write-Host ""
    Write-Host "--------------------------------------------------------------------------------" -ForegroundColor DarkCyan
    Write-Host " TRAUMATEAM LIVE COCKPIT ENDPOINTS" -ForegroundColor White
    Write-Host "--------------------------------------------------------------------------------" -ForegroundColor DarkCyan
    Write-Host "  Svelte Dispatch HUD : http://127.0.0.1:$UiPort" -ForegroundColor Green
    Write-Host "  Go HTTP / REST API  : http://127.0.0.1:$HttpPort" -ForegroundColor Green
    Write-Host "  Go WebSocket Stream : ws://127.0.0.1:$HttpPort/ws" -ForegroundColor Green
    Write-Host "  Go gRPC Endpoint    : 127.0.0.1:$GrpcPort" -ForegroundColor Green
    Write-Host "--------------------------------------------------------------------------------" -ForegroundColor DarkCyan
    Write-Host "Press Ctrl+C to stop the Vite dev server and background Go gateway." -ForegroundColor Yellow
    Write-Host ""

    Push-Location $FrontendDir
    try {
        if (-not (Test-Path (Join-Path $FrontendDir "node_modules"))) {
            Write-Step "Installing frontend dependencies (npm ci)..."
            & npm ci
            if ($LASTEXITCODE -ne 0) {
                throw "npm ci failed with exit code $LASTEXITCODE."
            }
        }
        & npm run dev '--' --host 127.0.0.1 --port $UiPort
        if ($LASTEXITCODE -ne 0) {
            throw "Svelte dev server exited with code $LASTEXITCODE."
        }
    } finally {
        Pop-Location
    }
} finally {
    if ($null -ne $GatewayProc -and -not $GatewayProc.HasExited) {
        Write-Step "Stopping background Go gateway (PID $($GatewayProc.Id))..."
        Stop-Process -Id $GatewayProc.Id -Force -ErrorAction SilentlyContinue
        $GatewayProc.WaitForExit(3000) | Out-Null
        Write-Pass "Background Go gateway stopped cleanly."
    }
    Remove-Item -Path $StdoutLog, $StderrLog -Force -ErrorAction SilentlyContinue
}
