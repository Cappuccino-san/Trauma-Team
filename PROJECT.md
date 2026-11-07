# Project: TraumaTeam Mission Feasibility Engine (Comprehensive Truth, Functionality & Test Audit)

## Architecture

The TraumaTeam mission-feasibility system evaluates critical operational requirements for Helicopter Air Ambulance (HAA) dispatch across 4 languages plus a standalone HTML console artifact:

1. **Rust RFM & Mass-Balance Subsystem (`crates/traumateam_rfm`)**:
   - Zero-cost dimensional newtypes (`Kilograms`, `Millimeters`, `KilogramMillimeters`, `Feet`, `Celsius`, `Knots`, `KgPerHour`) with compile-time dimensional operators.
   - Bilinear and trilinear zero-allocation grid interpolators for aircraft performance (IGE, OGE, OEI Cat A, Cruise TAS, and Fuel Flow) with panic-free NaN boundary guards.
   - OEM airframe models: Airbus H145 (BK117 D-2) and Bell 429 GlobalRanger.
   - Static moment summation, dynamic piecewise linear fuel depletion schedule lookups, and certified center-of-gravity envelope interpolation and margin checking.
   - C-ABI FFI exports (`libtraumateam_rfm.so`) consumed by Go via `cgo`.

2. **Go Weather Ingestion Service & WebSocket/gRPC Gateway (`services/weather_ingest`)**:
   - Ingestion and decoding of raw NOAA AWC data feeds: METAR/SPECI, TAF, Winds Aloft (FD), and SIGMETs (Convective & Non-convective).
   - Deterministic FAA 14 CFR Part 135 Subpart L compliance engine (evaluating § 135.609 Table 1 ceiling and visibility minimums across mountainous/non-mountainous, day/night, local/cross-country, NVIS/Autopilot equipment, OpSpec A021 relief, § 135.615 obstacle clearance, and § 135.611 IFR rules).
   - Native `cgo` FFI bridge (`pkg/kinematics`) calling `libtraumateam_rfm.so` for both Airbus H145 (`airframe_id: 0`) and Bell 429 (`airframe_id: 1`).
   - High-performance gRPC server (port 50051) and HTTP/WebSocket Gateway (port 8080) streaming real-time JSON updates and evaluation results.

3. **Python FAA 14 CFR Part 135 Subpart L Reference Engine (`traumateam/`, `tests/`)**:
   - Reference aerodynamic, atmospheric, wind-triangle, mass-and-balance, RFM interpolation (`scipy`), multi-leg mission integrator, and 14 CFR Part 135 Subpart L regulatory compliance engine (`far135_615.py` / § 135.609, `terrain_clearance.py` / § 135.615, `fuel_reserves.py` / §§ 135.209 & 135.223).
   - Evaluated via `pytest` and `hypothesis` property-based tests.

4. **TypeScript/Svelte 5 Avionics Dispatch Console (`frontend/`)**:
   - Connected to Go WebSocket gateway on port 8080 via Vite proxy on port 5173.
   - Sequential 01–05 panel layout (`CorridorControlPanel`, `CgEnvelope`, `RouteMap` with MapLibre GL & SIGMET polygons, `StationWeatherGrid`, `HazardAndTracePanel` with Audit Trace/SIGMETs/WS Envelopes/CLI Console), `VerdictBanner`, `ConnectionHeader`, and `CustomPresetModal` (JSON Upload & Visual Builder).

5. **Standalone Avionics Console Artifact (`traumateam_console.html`)**:
   - Self-contained HTML/JS/SVG console at `/home/antigravity/.gemini/jetski/brain/v1/default/jetski_remote_agents_3p/d4b2f0f0-39f5-4597-a8b2-48a9ee5b661e/traumateam_console.html` with interactive atmospheric/kinematics sliders, OEM Mass & CG envelope plot (H145 & Bell 429), route map, METAR table, deterministic audit trace, SIGMETs, custom JSON preset modal, and CLI console.

---

## Feature Inventory

Every feature discovered from the Phase 0 Survey across all 5 components is inventoried below and assigned to M4 (Truth, Functionality & Edge-Case Test Remediation):

| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Rust Dimensional Newtypes & Interpolators | `#[repr(transparent)]` f64 unit wrappers, 1D/2D/3D zero-allocation interpolators with NaN guards | M1 (Verified) | survey_1 |
| 2 | Rust OEM Airframes & C-FFI Exports | H145 & Bell 429 RFM/CG datasets and C-ABI FFI functions (`traumateam_eval_mission_kinematics_ffi`, etc.) | M1 (Verified) | survey_1 |
| 3 | Go NOAA Parsers & gRPC/WS Gateway | METAR, TAF, Winds Aloft, SIGMET parsers; gRPC `:50051` and HTTP/WS `:8080` servers | M2 (Verified) | survey_1 |
| 4 | Go CGO FFI Bell 429 Default Payloads & Linux rpath | Fix `handleEvaluateKinematics` (`client.go:252-260`) Bell 429 default station IDs (`status code -7`), unify `main.go`/`cmd/server/main.go` station lists, and add `-Wl,-rpath` to `ffi.go:8` | M4 | survey_1, survey_3 |
| 5 | Python Bell 429 Dispatch & Mission Integrator Truth | Fix `decision_engine.py:77,88` hardcoded H145 station check (`KeyError` on Bell 429), `terrain_clearance.py:33-51` cross-waypoint obstacle summation & § 135.615(b) clearance alignment, and `mission_integrator.py:222,335` mid-mission fuel exhaustion & reserve altitude | M4 | survey_2 |
| 6 | Python & Go 14 CFR Part 135 Subpart L Citations & Edge-Case Tests | Clarify § 135.609 Table 1 vs § 135.615 citations, support `AutopilotEquipped` in Go § 135.609 Night credit, and add missing Python unit tests (Bell 429 E2E, terrain clearance, severe hazards, OEI Cat A deficit, unflyable wind, forward/left CG, IFR reserves) | M4 | survey_2 |
| 7 | Svelte 5 Frontend Truth & WebSocket Kinematics | Wire `EVALUATE_KINEMATICS` in `websocket.ts` & `CgEnvelope.svelte`, support true Bell 429 OEM CG envelope (`[3120..3480] mm`, MTOW `3175 kg`) + live TAS/FF/ETE updates, align `NOMINAL_GO` fixture wind gust with backend, fix CLI `eval` offline fallback (`isEvaluating` stuck), clear `activeHazards` on preset switch, and fix a11y warnings | M4 | survey_3 |
| 8 | Standalone Console (`traumateam_console.html`) Runtime & OEM Truth | Fix critical `hazardCountBadge` null `TypeError` crash at line 855, replace fictitious `[4560, 4820] mm` CG limits with true OEM H145 (`[3780..4200] mm`, empty arm `3910 mm`) and Bell 429 (`[3120..3480] mm`, empty arm `3340 mm`) envelopes, wire `airframeSelect` event listener, and add automated verification tests | M4 | survey_3 |

---

## Milestones

| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Rust RFM & Mass Balance Crate | `crates/traumateam_rfm`: zero-cost abstractions, airframes, moment summation, CG envelope checks, and cargo test parity (61 tests pass) | none | DONE |
| M2 | Go Weather Service & WebSocket Gateway | `services/weather_ingest`: NOAA feed parsers, FAA Part 135.L evaluator, Protobuf compilation, gRPC server, and WebSocket broadcast gateway (92 tests pass) | none | DONE |
| M3 | E2E Testing & Acceptance Verification | Project-wide integration: zero-network test suite execution (57 E2E tests pass), client streaming verification | M1, M2 | DONE |
| M4 | Full-Stack Truth, Functionality & Test Audit Remediation | Fix all 9 bugs/discrepancies across Go (`pkg/gateway/client.go`, `pkg/kinematics/ffi.go`, `pkg/regulations/far135_609.go`, `cmd/server/main.go`), Python (`traumateam/` & `tests/`), Svelte 5 Frontend (`frontend/`), and Standalone HTML Console (`traumateam_console.html`), plus comprehensive new unit & E2E tests | M1, M2, M3 | IN_PROGRESS |

---

## Interface Contracts

### Go Weather Ingest Service ↔ Frontend / Svelte Client (WebSocket)
- **Protocol**: WebSocket over TCP (`ws://<host>:8080/ws`).
- **Client Actions**:
  - `SUBSCRIBE`: `{"action": "SUBSCRIBE", "request_id": "<id>", "params": {"stations": ["KAPA", "KLXV"]}}`
  - `EVALUATE_MINIMUMS`: `{"action": "EVALUATE_MINIMUMS", "request_id": "<id>", "params": {"corridor": {...}, "regulatory_context": {...}}}`
  - `EVALUATE_KINEMATICS`: `{"action": "EVALUATE_KINEMATICS", "request_id": "<id>", "params": {"airframe_id": 0|1, "fuel_kg": ..., "field_elev_ft": ..., "altimeter_inhg": ..., "oat_c": ..., "payloads": [...]}}`
  - `PING`: `{"action": "PING", "timestamp": <epoch_ms>}`
- **Server Events**:
  - `WEATHER_SNAPSHOT`, `WEATHER_UPDATE`: Streamed weather data with METAR, TAF, winds aloft, and hazards.
  - `MINIMUMS_EVALUATION`: Streamed evaluation response with `verdict`, `limiting_factor`, margins, and legal citations.
  - `KINEMATICS_EVALUATION`: Streamed Rust FFI kinematics, atmosphere, hover ceilings, cruise TAS/FF, and Mass & CG evaluation.
  - `PONG`: Heartbeat response.

### Go Weather Ingest Service ↔ External Systems (gRPC)
- **Interface**: `weather.WeatherIngestService`
- **RPCs**:
  - `GetCurrentConditions(StationConditionsRequest) -> CurrentConditionsResponse`
  - `StreamWeatherFeeds(StreamWeatherRequest) -> stream WeatherFeedUpdate`
  - `EvaluateFlightMinimums(FlightMinimumsEvaluationRequest) -> FlightMinimumsEvaluationResponse`
  - `StreamFlightMinimums(StreamFlightMinimumsRequest) -> stream FlightMinimumsUpdate`

---

## Code Layout

- `crates/traumateam_rfm/`: Rust RFM & Mass-Balance engine (`src/*.rs`, `tests/*.rs`, `Cargo.toml`)
- `services/weather_ingest/`: Go Weather Ingest, gRPC, CGO FFI & WebSocket Gateway (`main.go`, `cmd/server/main.go`, `proto/weather.proto`, `pkg/**/*.go`, `test/**/*.go`)
- `traumateam/` & `tests/`: Python reference engine & pytest/Hypothesis test suite (`traumateam/**/*.py`, `tests/test_*.py`, `demo_dispatch.py`)
- `frontend/`: TypeScript/Svelte 5 Avionics Dispatch Console (`src/**/*.{ts,svelte}`, `tests/**/*.test.ts`)
- `/home/antigravity/.gemini/jetski/brain/v1/default/jetski_remote_agents_3p/d4b2f0f0-39f5-4597-a8b2-48a9ee5b661e/traumateam_console.html`: Standalone HTML avionics console artifact
