#!/usr/bin/env bash
# ==============================================================================
# TraumaTeam Single-Command Local Launch & Deterministic Smoke-Check Script
# ==============================================================================
set -euo pipefail

CHECK_ONLY=0
HTTP_PORT="${HTTP_PORT:-8080}"
GRPC_PORT="${GRPC_PORT:-50051}"
UI_PORT="${UI_PORT:-5173}"

usage() {
  cat <<'EOF'
Usage: ./start_demo.sh [--check-only]

Options:
  --check-only   Build Rust cdylib, Go gateway, and Svelte production bundle,
                 run /healthz, /api/v1/conditions, /api/v1/kinematics, and /ws
                 PING/PONG smoke checks, then stop the gateway cleanly.
  -h, --help     Show this help message.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --check-only)
      CHECK_ONLY=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "[ERROR] Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUST_DIR="$REPO_ROOT/crates/traumateam_rfm"
RUST_RELEASE_DIR="$RUST_DIR/target/release"
GO_DIR="$REPO_ROOT/services/weather_ingest"
FRONTEND_DIR="$REPO_ROOT/frontend"

if [[ -d "$HOME/.cargo/bin" && ":$PATH:" != *":$HOME/.cargo/bin:"* ]]; then
  export PATH="$HOME/.cargo/bin:$PATH"
fi

for cmd in cargo go npm; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "[ERROR] Required toolchain command '$cmd' was not found on PATH." >&2
    exit 1
  fi
done

OS_NAME="$(uname -s)"
case "$OS_NAME" in
  Darwin)
    LIB_FILE="libtraumateam_rfm.dylib"
    GO_BIN="weather_ingest"
    USE_CGO=1
    ;;
  MINGW*|MSYS*|CYGWIN*)
    LIB_FILE="traumateam_rfm.dll"
    GO_BIN="weather_ingest.exe"
    USE_CGO=0
    ;;
  *)
    LIB_FILE="libtraumateam_rfm.so"
    GO_BIN="weather_ingest"
    USE_CGO=1
    ;;
esac

RUST_LIB_PATH="$RUST_RELEASE_DIR/$LIB_FILE"
GO_BIN_PATH="$GO_DIR/$GO_BIN"
GATEWAY_PID=""
LOG_FILE="$(mktemp "${TMPDIR:-/tmp}/traumateam_gateway.XXXXXX.log")"

cleanup() {
  if [[ -n "$GATEWAY_PID" ]] && kill -0 "$GATEWAY_PID" 2>/dev/null; then
    echo "[+] Stopping background Go gateway (PID $GATEWAY_PID)..."
    kill "$GATEWAY_PID" 2>/dev/null || true
    for _ in {1..20}; do
      if ! kill -0 "$GATEWAY_PID" 2>/dev/null; then
        break
      fi
      sleep 0.1
    done
    if kill -0 "$GATEWAY_PID" 2>/dev/null; then
      kill -9 "$GATEWAY_PID" 2>/dev/null || true
    fi
    wait "$GATEWAY_PID" 2>/dev/null || true
    echo "[PASS] Background Go gateway stopped cleanly."
  fi
  rm -f "$LOG_FILE"
}
trap cleanup EXIT INT TERM

echo "================================================================================"
echo " TRAUMATEAM 4-TIER DEMO LAUNCHER & VERIFICATION HARNESS ($OS_NAME)"
echo "================================================================================"

# 1. Build Rust shared library (cdylib)
echo "[+] Building Rust RFM physics core (cargo build --release)..."
(cd "$RUST_DIR" && cargo build --release)
if [[ ! -f "$RUST_LIB_PATH" ]]; then
  echo "[ERROR] Expected Rust shared library not found at: $RUST_LIB_PATH" >&2
  exit 1
fi
echo "[PASS] Rust cdylib ready: $RUST_LIB_PATH"

# 2. Configure FFI library environment variables
export TRAUMATEAM_RFM_LIB_PATH="$RUST_LIB_PATH"
export LD_LIBRARY_PATH="$RUST_RELEASE_DIR${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
export DYLD_LIBRARY_PATH="$RUST_RELEASE_DIR${DYLD_LIBRARY_PATH:+:$DYLD_LIBRARY_PATH}"
export PATH="$RUST_RELEASE_DIR:$PATH"
echo "[+] Configured TRAUMATEAM_RFM_LIB_PATH=$TRAUMATEAM_RFM_LIB_PATH"

# 3. Build Go weather_ingest gateway
echo "[+] Building Go telemetry & compliance gateway ($GO_BIN)..."
if [[ "$USE_CGO" -eq 1 ]]; then
  (cd "$GO_DIR" && CGO_ENABLED=1 go build -o "$GO_BIN" main.go)
else
  (cd "$GO_DIR" && go build -o "$GO_BIN" main.go)
fi
echo "[PASS] Go gateway built: $GO_BIN_PATH"

# 4. In --check-only mode, build Svelte production bundle
if [[ "$CHECK_ONLY" -eq 1 ]]; then
  echo "[+] Building Svelte 5 frontend production bundle (npm run build)..."
  if [[ ! -d "$FRONTEND_DIR/node_modules" ]]; then
    echo "[+] Installing frontend dependencies (npm ci)..."
    (cd "$FRONTEND_DIR" && npm ci)
  fi
  (cd "$FRONTEND_DIR" && npm run build)
  echo "[PASS] Frontend production bundle built cleanly."
fi

# 5. Verify ports are free before launch
check_port_free() {
  local port="$1"
  if command -v python3 >/dev/null 2>&1; then
    python3 - "$port" <<'PY'
import socket, sys
port = int(sys.argv[1])
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.settimeout(0.3)
try:
    if s.connect_ex(("127.0.0.1", port)) == 0:
        sys.stderr.write(f"[ERROR] Port 127.0.0.1:{port} is already in use.\n")
        sys.exit(1)
finally:
    s.close()
PY
  elif command -v lsof >/dev/null 2>&1; then
    if lsof -iTCP:"$port" -sTCP:LISTEN -Pn >/dev/null 2>&1; then
      echo "[ERROR] Port 127.0.0.1:$port is already in use." >&2
      exit 1
    fi
  fi
}

check_port_free "$HTTP_PORT"
check_port_free "$GRPC_PORT"

# 6. Start Go gateway in background and poll /healthz until UP
echo "[+] Starting Go gateway on 127.0.0.1:$HTTP_PORT (HTTP/WS) and 127.0.0.1:$GRPC_PORT (gRPC)..."
(cd "$GO_DIR" && "$GO_BIN_PATH" -http-port "$HTTP_PORT" -grpc-port "$GRPC_PORT" >"$LOG_FILE" 2>&1) &
GATEWAY_PID=$!

BASE_HTTP_URL="http://127.0.0.1:$HTTP_PORT"
HEALTH_URL="$BASE_HTTP_URL/healthz"
IS_HEALTHY=0

for _ in $(seq 1 80); do
  if ! kill -0 "$GATEWAY_PID" 2>/dev/null; then
    echo "[ERROR] Go gateway exited prematurely. Logs:" >&2
    cat "$LOG_FILE" >&2 || true
    exit 1
  fi

  if command -v curl >/dev/null 2>&1; then
    if resp="$(curl -fsS --max-time 2 "$HEALTH_URL" 2>/dev/null)" && [[ "$resp" == *'"UP"'* ]]; then
      IS_HEALTHY=1
      break
    fi
  elif command -v python3 >/dev/null 2>&1; then
    if python3 - "$HEALTH_URL" <<'PY' 2>/dev/null
import json, sys, urllib.request
with urllib.request.urlopen(sys.argv[1], timeout=2) as r:
    data = json.loads(r.read().decode("utf-8"))
    sys.exit(0 if r.status == 200 and data.get("status") == "UP" else 1)
PY
    then
      IS_HEALTHY=1
      break
    fi
  fi
  sleep 0.25
done

if [[ "$IS_HEALTHY" -ne 1 ]]; then
  echo "[ERROR] Timed out waiting for $HEALTH_URL to report status=UP. Logs:" >&2
  cat "$LOG_FILE" >&2 || true
  exit 1
fi

# 7. Run deterministic REST + WebSocket smoke checks
if command -v python3 >/dev/null 2>&1; then
  python3 - "$HTTP_PORT" <<'PY'
import base64
import hashlib
import json
import os
import socket
import struct
import sys
import time
import urllib.request

port = int(sys.argv[1])
base = f"http://127.0.0.1:{port}"

def get_json(path: str) -> dict:
    with urllib.request.urlopen(base + path, timeout=5) as resp:
        if resp.status != 200:
            raise RuntimeError(f"GET {path} returned HTTP {resp.status}")
        return json.loads(resp.read().decode("utf-8"))

health = get_json("/healthz")
assert health.get("status") == "UP", f"Unexpected /healthz payload: {health}"
print(f"[PASS] Healthcheck UP at {base}/healthz (service={health.get('service')})")

cond = get_json("/api/v1/conditions?station=KLXV")
assert cond.get("station_id") == "KLXV", f"Unexpected station_id: {cond}"
metar = cond.get("metar") or {}
assert metar.get("station_id") == "KLXV", f"Missing METAR for KLXV: {cond}"
print(f"[PASS] REST /api/v1/conditions?station=KLXV OK (Temp={metar.get('temperature_c')}C, QNH={metar.get('altimeter_inhg')} inHg)")

kin = get_json("/api/v1/kinematics?station=KLXV")
assert kin.get("airframe"), f"Missing airframe in kinematics: {kin}"
assert kin.get("verdict") in ("GO", "CAUTION", "NO-GO"), f"Invalid verdict: {kin}"
assert kin.get("pressure_altitude_ft", 0) > 0, f"Invalid PA: {kin}"
assert kin.get("max_oge_hover_weight_kg", 0) > 0, f"Invalid max OGE: {kin}"
mb = kin.get("mass_balance") or {}
assert mb.get("gross_weight_kg", 0) > 0, f"Invalid mass_balance: {kin}"
print(
    f"[PASS] REST /api/v1/kinematics?station=KLXV OK "
    f"(Airframe={kin['airframe']}, Verdict={kin['verdict']}, "
    f"PA={kin['pressure_altitude_ft']:.0f} ft, DA={kin['density_altitude_ft']:.0f} ft, "
    f"OGE Margin={kin['oge_margin_kg']:.1f} kg)"
)

# WebSocket RFC 6455 handshake + PING/PONG check using Python stdlib
key = base64.b64encode(os.urandom(16)).decode("ascii")
expected_accept = base64.b64encode(
    hashlib.sha1((key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").encode("ascii")).digest()
).decode("ascii")

sock = socket.create_connection(("127.0.0.1", port), timeout=5)
try:
    req = (
        f"GET /ws HTTP/1.1\r\n"
        f"Host: 127.0.0.1:{port}\r\n"
        f"Upgrade: websocket\r\n"
        f"Connection: Upgrade\r\n"
        f"Sec-WebSocket-Key: {key}\r\n"
        f"Sec-WebSocket-Version: 13\r\n\r\n"
    )
    sock.sendall(req.encode("ascii"))

    buf = b""
    while b"\r\n\r\n" not in buf:
        chunk = sock.recv(4096)
        if not chunk:
            raise RuntimeError("EOF during WebSocket handshake")
        buf += chunk
    header_bytes, leftover = buf.split(b"\r\n\r\n", 1)
    headers_text = header_bytes.decode("latin1")
    assert "101" in headers_text.splitlines()[0], f"WS handshake failed: {headers_text}"
    assert expected_accept in headers_text, f"Invalid Sec-WebSocket-Accept in: {headers_text}"

    req_id = f"smoke-ping-{int(time.time() * 1000)}"
    payload = json.dumps({"action": "PING", "request_id": req_id, "timestamp": int(time.time() * 1000)}).encode("utf-8")
    mask = os.urandom(4)
    length = len(payload)
    if length < 126:
        frame_hdr = struct.pack("!BB", 0x81, 0x80 | length)
    elif length < 65536:
        frame_hdr = struct.pack("!BBH", 0x81, 0x80 | 126, length)
    else:
        frame_hdr = struct.pack("!BBQ", 0x81, 0x80 | 127, length)
    masked = bytes(b ^ mask[i % 4] for i, b in enumerate(payload))
    sock.sendall(frame_hdr + mask + masked)

    def read_exact(n: int) -> bytes:
        nonlocal leftover
        while len(leftover) < n:
            more = sock.recv(4096)
            if not more:
                raise RuntimeError("EOF reading WebSocket frame")
            leftover += more
        out, leftover = leftover[:n], leftover[n:]
        return out

    b1, b2 = read_exact(2)
    opcode = b1 & 0x0F
    assert opcode == 0x1, f"Expected text WS frame, got opcode {opcode}"
    plen = b2 & 0x7F
    if plen == 126:
        (plen,) = struct.unpack("!H", read_exact(2))
    elif plen == 127:
        (plen,) = struct.unpack("!Q", read_exact(8))
    if b2 & 0x80:
        rmask = read_exact(4)
        raw_msg = bytes(b ^ rmask[i % 4] for i, b in enumerate(read_exact(plen)))
    else:
        raw_msg = read_exact(plen)

    pong = json.loads(raw_msg.decode("utf-8"))
    assert pong.get("event") == "PONG" and pong.get("request_id") == req_id, f"Unexpected WS reply: {pong}"
    print(f"[PASS] WebSocket ws://127.0.0.1:{port}/ws PING/PONG OK (seq={pong.get('seq')}, request_id={req_id})")
finally:
    sock.close()
PY
else
  echo "[PASS] Healthcheck UP at $HEALTH_URL"
  curl -fsS "$BASE_HTTP_URL/api/v1/conditions?station=KLXV" | grep -q '"KLXV"'
  echo "[PASS] REST /api/v1/conditions?station=KLXV OK"
  curl -fsS "$BASE_HTTP_URL/api/v1/kinematics?station=KLXV" | grep -q '"pressure_altitude_ft"'
  echo "[PASS] REST /api/v1/kinematics?station=KLXV OK"
  node -e "
    const ws = new WebSocket('ws://127.0.0.1:$HTTP_PORT/ws');
    const reqId = 'smoke-ping-' + Date.now();
    const timer = setTimeout(() => { console.error('WS timeout'); process.exit(1); }, 5000);
    ws.addEventListener('open', () => ws.send(JSON.stringify({ action: 'PING', request_id: reqId, timestamp: Date.now() })));
    ws.addEventListener('message', (ev) => {
      const msg = JSON.parse(ev.data);
      if (msg.event === 'PONG' && msg.request_id === reqId) {
        clearTimeout(timer);
        console.log('[PASS] WebSocket ws://127.0.0.1:$HTTP_PORT/ws PING/PONG OK (seq=' + msg.seq + ')');
        ws.close();
        process.exit(0);
      }
    });
    ws.addEventListener('error', (err) => { console.error(err); process.exit(1); });
  "
fi

if [[ "$CHECK_ONLY" -eq 1 ]]; then
  echo ""
  echo "[PASS] ALL CHECK-ONLY SMOKE CHECKS PASSED CLEANLY."
  exit 0
fi

# 8. Interactive mode: print service URLs and launch Svelte Vite dev server
echo ""
echo "--------------------------------------------------------------------------------"
echo " TRAUMATEAM LIVE COCKPIT ENDPOINTS"
echo "--------------------------------------------------------------------------------"
echo "  Svelte Dispatch HUD : http://127.0.0.1:$UI_PORT"
echo "  Go HTTP / REST API  : http://127.0.0.1:$HTTP_PORT"
echo "  Go WebSocket Stream : ws://127.0.0.1:$HTTP_PORT/ws"
echo "  Go gRPC Endpoint    : 127.0.0.1:$GRPC_PORT"
echo "--------------------------------------------------------------------------------"
echo "Press Ctrl+C to stop the Vite dev server and background Go gateway."
echo ""

if [[ ! -d "$FRONTEND_DIR/node_modules" ]]; then
  echo "[+] Installing frontend dependencies (npm ci)..."
  (cd "$FRONTEND_DIR" && npm ci)
fi
(cd "$FRONTEND_DIR" && npm run dev -- --host 127.0.0.1 --port "$UI_PORT")
