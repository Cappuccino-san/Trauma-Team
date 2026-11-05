# Project: TraumaTeam Mission Feasibility Engine

## Architecture

The TraumaTeam mission-feasibility system evaluates critical operational requirements for Helicopter Air Ambulance (HAA) dispatch. It consists of two primary subsystems:

1. **Rust RFM & Mass-Balance Subsystem (`crates/traumateam_rfm`)**:
   - Zero-cost dimensional newtypes (`Kilograms`, `Millimeters`, `KilogramMillimeters`, `Feet`, `Celsius`, `Knots`, `KgPerHour`) with compile-time dimensional operators.
   - Bilinear and trilinear zero-allocation grid interpolators for aircraft performance (IGE, OGE, OEI Cat A, Cruise TAS, and Fuel Flow) with panic-free NaN boundary guards.
   - OEM airframe models: Airbus H145 (BK117 D-2) and Bell 429 GlobalRanger.
   - Static moment summation, dynamic piecewise linear fuel depletion schedule lookups, and certified center-of-gravity envelope interpolation and margin checking.

2. **Go Weather Ingestion Service & WebSocket Gateway (`services/weather_ingest`)**:
   - Ingestion and decoding of raw NOAA AWC data feeds: METAR/SPECI, TAF, Winds Aloft (FD), and SIGMETs (Convective & Non-convective).
   - Deterministic FAA 14 CFR Part 135 Subpart L compliance engine (evaluating § 135.609 Table 1 ceiling and visibility minimums across mountainous/non-mountainous, day/night, local/cross-country, and NVG/HTAWS equipment, plus § 135.615 obstacle clearance and § 135.611 IFR rules).
   - High-performance gRPC server (port 50051) serving weather conditions, streaming feed updates, and evaluating flight minimums.
   - WebSocket Gateway and pub/sub broadcast Hub (port 8080) streaming real-time JSON updates and evaluation results to frontend Svelte clients.
   - 100% offline, zero-external-network testing harness utilizing embedded NOAA raw text fixtures and in-memory `bufconn` and `httptest`.

---

## Feature Inventory

Every feature discovered from the Survey phase has been implemented and verified:

| # | Feature | Description | Milestone | Status |
|---|---------|-------------|-----------|--------|
| 1 | Rust Dimensional Newtypes | Transparent f64 unit wrappers (`Kilograms`, `Millimeters`, etc.) with strict compile-time operator semantics | M1 | VERIFIED |
| 2 | Zero-Allocation Interpolators | 1D piecewise linear, 2D bilinear, and 3D trilinear stack interpolators with NaN guards matching SciPy `RegularGridInterpolator` | M1 | VERIFIED |
| 3 | Static Moment & CG Summation | Total gross weight, longitudinal/lateral static moments, fuel tank arm schedule interpolation | M1 | VERIFIED |
| 4 | CG Envelope Limit Enforcement | Dynamic piecewise linear envelope limit interpolation, margin calculation, and `LimitingFactor` exceedance flags | M1 | VERIFIED |
| 5 | OEM Airframe Implementations | Full certified tabular data for Airbus H145 and Bell 429 | M1 | VERIFIED |
| 6 | Rust Test Parity | Full numerical parity against Python tests (`test_rfm_performance.py`, `test_mass_balance.py`) via `cargo test` (57 tests pass) | M1 | VERIFIED |
| 7 | METAR Raw Parser | Zero-dependency parser for METAR/SPECI handling compound visibilities, obscurations (`VV`), clear sky variants, negative temps | M2 | VERIFIED |
| 8 | TAF Forecast Parser | Parser for TAF forecast periods (`FM`, `TEMPO`, `BECMG`, `PROB30`) for mission lookahead | M2 | VERIFIED |
| 9 | Winds Aloft Decoder & Vector Interp | Winds aloft parser (+50 offset, calm 9900) and trigonometric u/v interpolation across altitude layers | M2 | VERIFIED |
| 10 | SIGMET Parser & Hazard Detection | Convective SIGMET (`WST`) and non-convective SIGMET (`WS`) hazard detection and flight corridor intersection | M2 | VERIFIED |
| 11 | FAA Part 135 Subpart L Rule Matrix | Deterministic compliance engine for 14 CFR § 135.609 Table 1, § 135.615, and § 135.611 IFR rules | M2 | VERIFIED |
| 12 | Protobuf Schema & gRPC Service | Compiled `proto/weather.proto` and gRPC server implementing `WeatherIngestService` (`GetCurrentConditions`, `StreamWeatherFeeds`, `EvaluateFlightMinimums`, `StreamFlightMinimums`) | M2 | VERIFIED |
| 13 | WebSocket Gateway & Hub | Broadcast Hub with client read/write pumps, topic subscription (`station:*`, `corridor:*`), and slow-consumer backpressure protection | M2 | VERIFIED |
| 14 | WebSocket JSON Protocol | Bidirectional JSON envelope protocol for Svelte client requests and server event streaming | M2 | VERIFIED |
| 15 | Dual-Server Runtime | Unified process launching gRPC (:50051) and HTTP/WS (:8080) with coordinated graceful shutdown | M2 | VERIFIED |
| 16 | Zero-External-Network Test Suite | Automated unit and integration tests using embedded NOAA raw text fixtures, `bufconn`, and `httptest` without external calls | M3 | VERIFIED |
| 17 | Svelte Client Integration Compatibility | Compatibility verification for frontend WebSocket streaming and real-time flight minimum evaluation | M3 | VERIFIED |

---

## Milestones

| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Rust RFM & Mass Balance Crate | `crates/traumateam_rfm`: zero-cost abstractions, airframes, moment summation, CG envelope checks, and cargo test parity matching Python test suite (57 tests pass) | none | DONE |
| M2 | Go Weather Service & WebSocket Gateway | `services/weather_ingest`: NOAA feed parsers, FAA Part 135.L evaluator, Protobuf compilation, gRPC server, and WebSocket broadcast gateway (all package tests pass) | none | DONE |
| M3 | E2E Testing & Acceptance Verification | Project-wide integration: zero-network test suite execution (57 E2E tests pass), client streaming verification, and full acceptance criteria sign-off | M1, M2 | DONE |

---

## Interface Contracts

### Go Weather Ingest Service ↔ Frontend / Svelte Client (WebSocket)
- **Protocol**: WebSocket over TCP (`ws://<host>:8080/ws`).
- **Client Actions**:
  - `SUBSCRIBE`: `{"action": "SUBSCRIBE", "request_id": "<id>", "params": {"stations": ["KAPA", "KLXV"]}}`
  - `EVALUATE_MINIMUMS`: `{"action": "EVALUATE_MINIMUMS", "request_id": "<id>", "params": {"corridor": {...}, "regulatory_context": {...}}}`
  - `PING`: `{"action": "PING", "timestamp": <epoch_ms>}`
- **Server Events**:
  - `WEATHER_SNAPSHOT`, `WEATHER_UPDATE`: Streamed weather data with METAR, TAF, winds aloft, and hazards.
  - `MINIMUMS_EVALUATION`: Streamed evaluation response with `verdict`, `limiting_factor`, margins, and legal citations.
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

- `crates/traumateam_rfm/`:
  - `src/lib.rs`
  - `src/units.rs`
  - `src/interpolator.rs`
  - `src/rfm.rs`
  - `src/mass_balance.rs`
  - `src/airframes/*.rs`
  - `tests/*.rs`
  - `Cargo.toml`
- `services/weather_ingest/`:
  - `cmd/server/main.go`
  - `proto/weather.proto`
  - `pkg/pb/*.go`
  - `pkg/models/*.go`
  - `pkg/noaa/*.go`
  - `pkg/regulations/*.go`
  - `pkg/grpc/*.go`
  - `pkg/gateway/*.go`
  - `test/integration_test.go`
  - `test/e2e/*.go`
  - `go.mod`, `go.sum`
- `tests/`: Existing Python tests (read-only reference).
