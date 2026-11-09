# TraumaTeam Dispatch System: Architecture Specification & EGM Audit

**Document Status**: Active / Elephant Specification  
**Architecture Version**: 1.2.0  
**Scope**: 4-Language High-Integrity HEMS Dispatch & Kinematics Engine  
**Standards**: 14 CFR Part 135 Subpart L (§§ 135.609, 135.611, 135.615), OEM Airbus H145 RFM, OEM Bell 429 RFM  

---

## 1. System Topology & Data Flow

```
                      +-----------------------------+
                      |   NOAA AWC METAR / TAF Feed |
                      +--------------+--------------+
                                     | (HTTP Poller)
                                     v
+------------------------+   +------------------------------------+   +-----------------------+
|  TypeScript / Svelte 5 |<->|      Go Gateway Service (:8080)    |<->| Python Reference      |
|  Dispatcher UI (:5173) |WS |   - 14 CFR 135.609 Rule Engine     |   | Engine (traumateam/)  |
|  - 01 Regulatory Check |   |   - Route Corridor Hazard Evaluator|   | - Pytest parity suite |
|  - 02 Mass / CG Sliders|   |   - WebSocket Hub & Protocol Buffer|   +-----------------------+
|  - 03 RouteMap & HUD   |   +-----------------+------------------+
|  - 04 METAR / Hazards  |                     | (CGO In-Process FFI)
|  - 05 Raw Feed Monitor |                     v
|  - Trauma Console tab  |   +------------------------------------+
+------------------------+   |  Rust RFM & Kinematics Engine      |
                             |  - Bivariate Spline Interpolation  |
                             |  - High-DA Ceiling / HIGE / HOGE   |
                             |  - OEM CG Envelope Margin Calc     |
                             |  - Nan-safe Float Math             |
                             +------------------------------------+
```

---

## 2. Core Subsystems & Interface Contracts

### 2.1 Rust RFM Kinematics Engine (`crates/traumateam_rfm`)
- **Shared Object**: `libtraumateam_rfm.so`
- **C ABI (`services/weather_ingest/pkg/kinematics/traumateam_rfm.h`)**:
  - `traumateam_eval_mission_kinematics_ffi(const FfiWeatherMissionInput* input, const FfiPayloadItem* payloads, size_t payload_count, FfiMissionEvaluationResult* out_result) -> int32_t`
  - Zero-heap allocation contract using contiguous stack-friendly structs.
- **Invariants**:
  - Zero panic guarantee on `NaN`, `+Inf`, `-Inf`, or subnormal floats (`math.isfinite()` and IEEE 754 guards).
  - Aerodynamic monotonicity: Hover ceiling decreases monotonically with increasing gross weight and temperature.
  - Authentic OEM Flight Manual Station Arms:
    - **Airbus H145**: Pilot/Copilot (2,420 mm), Crew Attendant Aft LH/RH (3,350 mm), Primary Patient Litter (3,820 mm), Medical Cabinet Rack (4,280 mm), Rescue Hoist (4,400 mm), Aft Cargo (4,800 mm).
    - **Bell 429**: Pilot/Copilot (2,180 mm), Crew Attendant LH (3,100 mm), Primary Patient Litter (3,550 mm), Medical Equipment Rack (3,900 mm), Aft Baggage (4,650 mm).
  - Piecewise fuel-arm schedule interpolation matching OEM flight manuals.

### 2.2 Go Weather Gateway & Rule Engine (`services/weather_ingest`)
- **Transport**:
  - HTTP REST: `:8080/healthz`, `:8080/api/v1/dispatch`, `:8080/api/v1/weather`
  - WebSocket: `:8080/ws` (bidirectional JSON envelopes)
  - gRPC: `:50051` (`DispatchService`)
- **Envelopes**:
  - `ClientEnvelope`: `{ action: "SUBSCRIBE" | "UNSUBSCRIBE" | "GET_CONDITIONS" | "EVALUATE_MINIMUMS" | "EVALUATE_KINEMATICS" | "INJECT_PRESET" | "PING", request_id: ..., params: { ... } }`
  - `ServerEnvelope`: `{ event: "WEATHER_SNAPSHOT" | "WEATHER_UPDATE" | "MINIMUMS_EVALUATION" | "KINEMATICS_EVALUATION" | "HAZARD_ALERT" | "PONG" | "ACK" | "ERROR", request_id: ..., seq: ..., payload: ..., error: ... }`
- **Regulatory Rules (14 CFR § 135.609 Table 1)**:
  - VFR Ceiling and Visibility Minimums in Class G Airspace (Table 1: Non-Mountainous vs Mountainous, Day vs Night, Local vs Cross-Country, and Table 1 Column 2 relief with approved NVIS/HTAWS/Autopilot).
  - 14 CFR § 135.615 VFR Flight Planning (mandatory 300 ft day / 500 ft night obstacle clearance).

### 2.3 Svelte 5 / TypeScript Frontend (`frontend/src/`)
- **Component Layout (Sequential Panels 01 -> 05)**:
  - **Verdict Strip**: Full-width top banner displaying GO / CAUTION / NO-GO status, limiting factors, and FAR 135.609 status.
  - **01 // REGULATORY DISPATCH CRITERIA**: Flight conditions, FAR 135 criteria selection, day/night toggles.
  - **02 // ATMOSPHERE, MASS & CG ENVELOPE**: Dynamic atmospheric sliders, payload stations, gross weight progress bar, longitudinal CG balance track, and SVG envelope with real-time reticle.
  - **03 // AERONAUTICAL CORRIDOR & HAZARD GEODESIC**: MapLibre vector map (expandable 480px-680px), terrain elevation profile, obstacle clearance cones, dynamic HUD telemetry overlay.
  - **04 // CEILING & VISIBILITY HAZARDS**: METAR decode cards, density altitude alerts, mountain wave turbulence markers.
  - **05 // DISPATCH ENGINE DIAGNOSTIC LOG**: Raw telemetry JSON feed, WebSocket connectivity status, CGO FFI latency.
  - **TRAUMA CONSOLE**: Integrated modal/drawer with custom sim preset builder, JSON injector, and CLI console.

---

## 3. High-Value Audit Objectives for EGM

1. **Protocol & Data Contract Synchronization**:
   - Verify alignment across Go `ClientEnvelope`/`ServerEnvelope`, TypeScript `types.ts`, Rust FFI, and Python models.
   - Ensure custom preset injection correctly handles all fields without silent drops.
2. **Defensive Error Handling & Fault Isolation**:
   - Resilience against corrupt NOAA feeds, malformed coordinates, non-numeric weather values, or extreme DA conditions.
3. **Regulatory Truth**:
   - Verify exact compliance with 14 CFR § 135.609 Table 1 across all 8 operational quadrants (Mountainous x Day/Night x Local/X-Country).
4. **Performance & Memory Stability**:
   - Prevent unbounded WebSocket buffer growth during long-running streaming or reconnection storms.
5. **Stateful Reactive Consistency**:
   - Ensure UI sliders and custom preset uploads trigger full recalculations across all 5 panels immediately.
