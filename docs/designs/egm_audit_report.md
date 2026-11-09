# EGM Audit Report: TraumaTeam Dispatch & Kinematics Engine
**Auditor**: Independent Goldfish Reviewer  
**Methodology**: Dave Rensin's Elephant-Goldfish Model (EGM)  
**Target Codebase**: `/home/antigravity/Cappuccino-san/Trauma-Team`  
**Date**: November 9, 2026  
**Status**: Comprehensive Engineering Audit Completed  

---

## Executive Summary

An exhaustive, adversarial, and uncompromising audit was conducted across the 4 languages of the TraumaTeam HEMS (Helicopter Emergency Medical Services) dispatch system:
1. **Rust RFM Engine** (`crates/traumateam_rfm`)
2. **Go Gateway & WebSocket Hub** (`services/weather_ingest`)
3. **Python Reference Engine** (`traumateam/`, `tests/`)
4. **TypeScript/Svelte Frontend** (`frontend/src/`) and standalone console interface

The audit evaluated compliance against **14 CFR Part 135 Subpart L** (§§ 135.609, 135.611, 135.615), **OEM Rotorcraft Flight Manuals** (Airbus H145 BK117 D-2 and Bell 429 GlobalRanger), and the Elephant architectural specification (`docs/designs/egm_audit_and_improvements.md`).

While the core aerodynamic interpolator and CGO FFI bindings demonstrate robust numerical performance and zero-allocation mechanics, critical architectural drifts, race conditions, false-positive legal dispatches, and statutory citation confusions exist across the system boundaries.

---

## Phase 1: Strategic Altitude / Initial Zoom Out

### 1.1 Regulatory Alignment with 14 CFR Part 135 Subpart L
* **Statutory Citation Confusion**:
  * **14 CFR § 135.609 Table 1** is the sole statutory authority governing *VFR Ceiling and Visibility Requirements* for Helicopter Air Ambulance (HAA) operations in Class G airspace.
  * **14 CFR § 135.615** governs *VFR Flight Planning* (hazard/obstacle clearance minimums of 300 ft day / 500 ft night, alternate landing site identification, fuel reserves).
  * In the **Python engine** (`far135_615.py`) and **Rust engine** (`LimitingFactor::Far135_615CeilingBelowMinimums`, `LimitingFactor::Far135_615VisibilityBelowMinimums`), Table 1 minimums are misattributed to § 135.615 instead of § 135.609.
  * In the **Go gateway**, the rules are correctly attributed to `far135_609.go`, creating an inter-service regulatory dissonance.
* **Daytime NVG/HTAWS Relief Conflation**:
  * Under 14 CFR § 135.609 Table 1, Column 2 relief (500 ft ceiling / 2 SM visibility) is statutory *strictly for night operations* with approved HTAWS/NVIS or autopilot.
  * Table 1 does *not* provide daytime local relief down to 500-2; day local minimums are standard 800-2 unless authorized under an FAA Certificate of Waiver or Operational Specification (e.g., OpSpec A021).
  * The Python engine conflates NVG equipment with day relief unconditionally, forcing Go to introduce an ad-hoc flag `AllowOpSpecA021Relief` to maintain test parity.

### 1.2 OEM Rotorcraft Flight Manual (RFM) Fidelity
* **Elephant Specification Hallucination (Section 2.1)**:
  * The Elephant architectural specification (`egm_audit_and_improvements.md`) claims that both Airbus H145 and Bell 429 share the exact same longitudinal station arms: `PILOT_COPILOT` (1,850 mm), `PAX_FWD` (2,750 mm), `PAX_AFT` (3,400 mm), `MEDICAL_EQUIPMENT` (3,900 mm), `AFT_CARGO` (4,400 mm).
  * This is factually erroneous and physically impossible across differing airframe geometries.
  * In actual code, the Rust (`h145.rs`, `bell429.rs`), Python (`h145.py`, `bell429.py`), and TypeScript (`CgEnvelope.svelte`) implementations correctly adhere to OEM flight manual station datums:
    * **Airbus H145**: Pilot (2,420 mm), Attendant Aft (3,350 mm), Primary Litter (3,820 mm), Medical Cabinet (4,280 mm).
    * **Bell 429**: Pilot (2,180 mm), Attendant (3,100 mm), Primary Litter (3,550 mm), Equipment Rack (3,900 mm), Baggage (4,650 mm).
  * **Finding**: The Elephant specification document suffered from copy-paste fiction that does not reflect certified OEM aerodynamics or the actual codebase.

### 1.3 Identification of YAGNI Bloat
* **Dual WebSocket Hub Implementations in Go**:
  * `services/weather_ingest/pkg/websocket` contains a basic WebSocket hub and client (`hub.go`, `client.go`).
  * `services/weather_ingest/pkg/gateway` contains an entirely separate, feature-rich WebSocket hub and client (`hub.go`, `client.go`, `envelope.go`).
  * `pkg/websocket` is dead code in production, retained only by leftover integration test imports.
* **Duplicate Server Main Entry Points**:
  * `services/weather_ingest/main.go` and `services/weather_ingest/cmd/server/main.go` are near-identical, duplicate main packages competing for build targets.
* **Synthetic Feeds vs Claimed Live Poller**:
  * The specification topology asserts a live "NOAA AWC METAR / TAF Feed (HTTP Poller)".
  * In reality, there is zero external outbound network polling; `WeatherFeedClient` is an in-memory dictionary initialized with static fixtures, and the poller simply re-broadcasts static memory every 30 seconds.

---

## Phase 2: Comprehension Check

### 2.1 Interface Boundaries & End-to-End Data Flow

```
+-------------------------------------------------------------------------+
| SVELTE 5 FRONTEND (frontend/src/)                                       |
| - Controls: 01 Corridor, 02 Sliders, 03 RouteMap, 04 Weather, 05 Console|
| - Store: wsRouter sends ClientEnvelope -> ws.send()                     |
+------------------------------------+------------------------------------+
                                     | WebSocket JSON
                                     v
+-------------------------------------------------------------------------+
| GO GATEWAY (:8080 /ws, :50051 gRPC) (services/weather_ingest)           |
| - client.go: readPump() -> handleClientAction()                         |
| - Evaluator: regulations.EvaluateCorridorCompliance()                   |
| - CGO FFI Bridge: kinematics.EvaluateMissionKinematics()                |
+------------------------------------+------------------------------------+
                                     | C ABI (In-Process FFI, 0-Allocation)
                                     v
+-------------------------------------------------------------------------+
| RUST KINEMATICS ENGINE (crates/traumateam_rfm)                          |
| - traumateam_eval_mission_kinematics_ffi(in_ptr, payloads_ptr, out_ptr)|
| - ICAO Doc 7488/3 thermodynamic PA/DA computation                       |
| - 2D/3D Bivariate Spline Interpolation (HIGE, HOGE, OEI Cat A, Cruise)  |
| - Closed-form Wind Triangle & 14 CFR § 135.209 fuel reserve calculation |
+-------------------------------------------------------------------------+
```

### 2.2 Protocol Envelope Mismatch (Elephant Spec vs Reality)
* **Elephant Document Contract**:
  * Client: `{ type: "request_dispatch" | "subscribe_weather" | "inject_preset", payload: ... }`
  * Server: `{ type: "dispatch_verdict" | "weather_update" | "hazard_alert" | "error", data: ..., timestamp: ... }`
* **Actual Go / TypeScript Contract**:
  * Client: `{ action: "SUBSCRIBE" | "UNSUBSCRIBE" | "GET_CONDITIONS" | "EVALUATE_MINIMUMS" | "EVALUATE_KINEMATICS" | "PING", request_id: ..., params: { ... } }`
  * Server: `{ event: "WEATHER_SNAPSHOT" | "WEATHER_UPDATE" | "MINIMUMS_EVALUATION" | "KINEMATICS_EVALUATION" | "HAZARD_ALERT" | "PONG" | "ACK" | "ERROR", seq: ..., timestamp_utc: ..., payload: ..., error: ... }`
* **Architectural Drift**:
  * Field names (`type` vs `action`/`event`, `data` vs `payload`, `timestamp` vs `timestamp_utc`) are completely desynchronized between the Elephant document and the codebase.
  * The claimed `inject_preset` action does not exist on the Go gateway. Custom presets are instead intercepted client-side by `CustomPresetModal.svelte` and fed directly into Svelte stores (`wsRouter.routeEnvelope()`), bypassing the server entirely.

---

## Phase 3: Design Critique & Adversarial Probe

### 3.1 Concurrency Race & Panic in Go WebSocket Hub (`pkg/gateway/hub.go`)
* **Unsafe Concurrent Map Mutation**:
  In `hub.go` lines 64-77, inside the `case item := <-h.broadcast:` block:
  ```go
  h.mu.RLock()
  for client := range h.clients {
      if client.IsSubscribed(item.topic) {
          select {
          case client.send <- item.message:
          default:
              // Slow consumer drop
              close(client.send)
              delete(h.clients, client) // <-- CRITICAL DEFECT
          }
      }
  }
  h.mu.RUnlock()
  ```
  `delete(h.clients, client)` modifies `h.clients` while holding only a **read lock** (`RLock()`). Under high broadcast frequency or slow network clients, this causes a fatal Go runtime panic (`concurrent map iteration and map write`).
* **Double Channel Close Panic**:
  Closing `client.send` inside the broadcast loop while `client.readPump()` also triggers `unregister <- client` causes `close(client.send)` to be called twice, triggering `panic: close of closed channel`.

### 3.2 False "GO" / APPROVED Dispatch on Empty or Corrupt Corridors
* In `services/weather_ingest/pkg/regulations/evaluator.go`:
  ```go
  worstCeil := math.Inf(1)
  worstVis := math.Inf(1)
  ...
  if math.IsInf(worstCeil, 1) {
      worstCeil = 25000.0
  }
  if math.IsInf(worstVis, 1) {
      worstVis = 10.0
  }
  ```
  If a user inputs unrecognized, misspelled, or blank station identifiers, `stations` is empty. The evaluator defaults `worstCeil` to **25,000 ft** and `worstVis` to **10.0 SM**, outputting an **APPROVED / GO** verdict (`is_compliant: true`, margin +24,200 ft).
* **Life Safety Risk**: A flight corridor with zero valid weather reporting must fail closed (NO-GO / DISQUALIFIED under § 135.609 / § 135.611), rather than reporting an uninhibited clearance.

### 3.3 Thermodynamic Density Altitude Discrepancy (Go vs Rust)
* **Go Engine** (`metar_parser.go:320`):
  Uses the standard linear rule-of-thumb:
  $$\text{DA} = \text{PA} + 118.8 \times (\text{OAT} - \text{ISA}_\text{temp})$$
* **Rust Engine** (`ffi.rs:326`):
  Uses exact nonlinear ICAO Doc 7488/3 standard thermodynamic equations:
  $$\rho = \frac{P_0 \left(1 - \frac{L \cdot \text{PA}}{T_0}\right)^{\frac{g}{RL}}}{R \cdot T_\text{K}}$$
  $$\text{DA} = \frac{T_0}{L} \left(1 - \left(\frac{\rho}{\rho_0}\right)^{\frac{1}{\frac{g}{RL} - 1}}\right)$$
* **Operational Consequence**:
  At Leadville (KLXV, 9,934 ft MSL) on a 25°C summer day, the linear rule deviates from thermodynamic reality by **over 450 feet**. A dispatcher inspecting Panel 04 (Weather Grid) sees one DA, while Panel 02 (Mass & CG / Hover Performance) computes margins on another DA.

### 3.4 Python Reference Engine NaN and Negative Weight Vulnerability
* In `traumateam/performance/mass_balance.py`:
  If `fuel_kg = float('nan')` or payload weights are `NaN`:
  `gw > airframe.mtow_kg` evaluates to `False`.  
  `gw < airframe.min_flight_weight_kg` evaluates to `False`.  
  Both `MTOW_EXCEEDED` and `MIN_WEIGHT_VIOLATION` are bypassed!
  The engine proceeds to compute CG coordinates (`nan mm`), and erroneously triggers `LONGITUDINAL_CG_AFT_EXCEEDED` stating `"Longitudinal CG (nan mm) is nan mm behind aft limit (nan mm)"`.
* Python fails to implement the rigorous `is_finite()` and non-negative guards present in Rust.

### 3.5 Cross-Airframe Station Incompatibility (Bell 429 Drift)
* Airframe station IDs are structurally distinct:
  * H145 has `CrewAttendantAftLh` (ID 2) and `MedicalCabinetRack` (ID 8).
  * Bell 429 has `CrewAttendantLh` (ID 4) and `MedicalEquipmentRack` (ID 9).
* In `CgEnvelope.svelte`, the author manually mapped payload stations based on airframe selection. However, if custom payloads or raw API requests submit Station ID 2 to a Bell 429 evaluation, Rust FFI rejects it with `-7` (`TraumaTeamError::UnknownStation`), and the UI silently drops the update.

### 3.6 Frontend Reconnection Failure & Hardcoded Geo Bottleneck
* **Disabled Auto-Reconnect**:
  `wsRouter` is instantiated with `autoReconnect: false`. In `ConnectionHeader.svelte`, the `toggleConnection` handler connects to the WebSocket but never enables `autoReconnect`. If the Go service restarts or drops a packet, the UI remains permanently in `DISCONNECTED` state until manual operator intervention.
* **Hardcoded Station Geo Bounds**:
  `RouteMap.svelte` hardcodes exactly 11 airports in `STATION_COORDS`, and Go's `evaluator.go` hardcodes 15. Any mission outside Colorado Metro or Boston/New England silently drops corridor lines and fails hazard intersection tests.

---

## Phase 4: Intermediate Zoom Out

### 4.1 System Complexity vs Operational Utility
* The 4-language polyglot architecture (Rust + Go + Python + TypeScript) provides distinct advantages:
  * **Rust**: Certified, deterministic, zero-allocation kinematic and spline physics.
  * **Go**: High-throughput concurrent network gateway and regulatory state machine.
  * **TypeScript/Svelte 5**: Responsive, reactive dispatch HUD.
* However, the **Python** tier is currently an orphaned reference island:
  * It does not parse live METAR text.
  * It cites § 135.615 instead of § 135.609.
  * It has subtle NaN bugs not present in Rust.
  * It is not invoked in production.
  * *Recommendation*: Retain Python strictly as an offline property-based validation oracle (Hypothesis test suite) or retire it in favor of native Rust/Go integration tests.

### 4.2 Missing Artifact: Standalone Console (`traumateam_console.html`)
* The prompt and architectural overview explicitly referenced a standalone dispatch console (`traumateam_console.html`).
* No such file existed in the repository; the interactive CLI was instead embedded as a sub-tab within Panel 05 (`HazardAndTracePanel.svelte`). Operators requiring a standalone, unbundled lightweight console currently lack this deliverable.

---

## Phase 5: Implementation Readiness & Actionable Improvements

The findings are prioritized below by operational and safety severity:

| ID | Severity | File / Subsystem | Defect Summary | Recommended Action |
|---|---|---|---|---|
| **EGM-01** | **CRITICAL** | `services/weather_ingest/pkg/gateway/hub.go` | Concurrent map write under `RLock` and double channel closure on slow client drop in WebSocket Hub. | Replace `RLock()` with `Lock()` when unregistering/deleting slow clients, or route drops through the dedicated `unregister` channel. Protect `client.send` with a `sync.Once`. |
| **EGM-02** | **CRITICAL** | `services/weather_ingest/pkg/regulations/evaluator.go` | Empty or invalid station corridor defaults to 25,000 ft ceiling and 10 SM visibility, producing a false "APPROVED" / GO dispatch. | Explicitly check `if len(stations) == 0`. Return `DISQUALIFIED` / `NO-GO` with reason `"NO_WEATHER_REPORTING_AVAILABLE"` under 14 CFR § 135.609 / § 135.611. |
| **EGM-03** | **CRITICAL** | `traumateam/performance/mass_balance.py` | Missing `math.isnan()` and negative mass checks in Python reference engine causing silent NaN propagation and misleading error messages. | Add input validation guarding `not math.isfinite(fuel_kg) or fuel_kg < 0.0` and item weight checks matching Rust's `traumateam_rfm::mass_balance`. |
| **EGM-04** | **MAJOR** | `traumateam/`, `crates/traumateam_rfm/`, `services/weather_ingest/` | Statutory citation drift: VFR weather minimums misattributed to § 135.615 instead of § 135.609 Table 1 in Python and Rust limiting factor enums. | Unify all references across Rust (`Far135_609CeilingBelowMinimums`), Python, and Go to cite **14 CFR § 135.609 Table 1**, reserving § 135.615 for terrain/obstacle clearance. |
| **EGM-05** | **MAJOR** | `services/weather_ingest/pkg/noaa/metar_parser.go` vs `crates/traumateam_rfm/src/ffi.rs` | Density Altitude discrepancy: Go uses linear approximation while Rust FFI uses exact ICAO Doc 7488/3 equations, causing >400 ft divergence at high DA. | Deprecate linear DA in Go; delegate all Density Altitude computations in METAR processing to the linked Rust CGO FFI `ComputeAtmosphere`. |
| **EGM-06** | **MAJOR** | `docs/designs/egm_audit_and_improvements.md` & `CustomPresetModal.svelte` | Client-side-only preset injection; Elephant architectural specification claims server action `inject_preset`, but Go gateway has no such handler. | Implement `ActionInjectPreset` in Go `gateway/client.go` to store and broadcast custom presets across all connected clients via the Hub. Synchronize the Elephant spec to match real envelope field names (`action`, `event`, `params`, `payload`). |
| **EGM-07** | **MAJOR** | `services/weather_ingest/pkg/regulations/evaluator.go` & `frontend/src/lib/components/RouteMap.svelte` | Hardcoded 11-15 station coordinate map causes non-Colorado/non-Boston routes to lose map lines and hazard spatial intersection checks. | Integrate an ICAO station airport coordinate database (or FAA NASR dataset) into Go and frontend, with fallback geocoding. |
| **EGM-08** | **MAJOR** | Repository Root (`traumateam_console.html`) | Missing standalone console file specified in architectural deliverables. | Extract the interactive terminal emulator from `HazardAndTracePanel.svelte` into a self-contained, offline-capable `traumateam_console.html` artifact. |
| **EGM-09** | **MINOR** | `docs/designs/egm_audit_and_improvements.md` (Section 2.1) | Fictitious identical station arms listed for H145 and Bell 429 in Elephant specification. | Correct the Elephant design document to document real OEM arms (H145 pilot 2420mm vs Bell 429 pilot 2180mm). |
| **EGM-10** | **MINOR** | `services/weather_ingest/pkg/websocket` & `cmd/server/main.go` | Dead code and duplicate entry points (`pkg/websocket` vs `pkg/gateway`, and root `main.go` vs `cmd/server/main.go`). | Remove obsolete `pkg/websocket` package and consolidate root `main.go` to invoke `cmd/server/main.go`. |
| **EGM-11** | **MINOR** | `frontend/src/lib/websocket.ts` & `ConnectionHeader.svelte` | Automatic reconnection disabled by default; user connection drop leaves UI disconnected without retrying. | Enable `autoReconnect: true` when user clicks `LIVE WS`, ensuring exponential backoff reconnection takes effect. |
| **EGM-12** | **MINOR** | `services/weather_ingest/pkg/noaa/client.go` | `WeatherFeedClient.AddHazard` appends unbounded advisories to `hazardsCache` without eviction or expiration pruning. | Add periodic cleanup routine purging advisories where `ValidUntil.Before(now)`. |

---

## Conclusion

The TraumaTeam codebase demonstrates sophisticated aeronautical modeling, high-precision Rust kinematics, and an intuitive Svelte dispatch console. Addressing the concurrency race in the WebSocket Hub, preventing false-positive clearances on empty corridors, aligning Density Altitude thermodynamics across Go and Rust, and synchronizing the Elephant specification will elevate this system to certified mission-critical HEMS dispatch readiness.
