# E2E Test Infrastructure Specification: TraumaTeam Mission Feasibility Engine

**Document Version**: 1.0.0  
**Target Services**: `services/weather_ingest` (Go/gRPC + WebSocket Gateway) & Regulatory Compliance Subsystem  
**Author**: E2E Test Architect (`teamwork_test_writer_e2e`)  
**Status**: Ready / Implemented  
**Offline Integrity**: 100% Offline, Zero External Network Calls (`bufconn` + `httptest`)  

---

## 1. Executive Summary & Testing Philosophy

The TraumaTeam mission-feasibility system performs safety-critical go/no-go determinations for Helicopter Air Ambulance (HAA) operations. An erroneous flight release into deteriorating convective weather, below-minimum ceilings, or unmanageable mountain density altitudes can lead to Controlled Flight Into Terrain (CFIT) or catastrophic power deficits.

To guarantee zero-defect operational integrity, this E2E test infrastructure employs a **systematic 4-tier testing methodology**:
1. **Tier 1: Feature Coverage** — Verifies each independent functional capability in isolation with at least 5 distinct test cases per feature.
2. **Tier 2: Boundary & Corner Cases** — Stresses parser edge conditions, extreme numerical values, atypical meteorological codes, and corrupted data streams.
3. **Tier 3: Cross-Feature Combinations** — Evaluates interactions across multiple subsystems, including spatial SIGMET hazard intersection overriding surface METAR observations, multi-station corridor worst-case aggregation, and 14 CFR § 135.611 IFR transitions.
4. **Tier 4: Real-World Application Scenarios** — Validates full mission lifecycle simulations matching high-risk HAA operational theaters in the Colorado Rocky Mountains and Denver metropolitan area.

---

## 2. In-Memory Offline Test Architecture

All automated E2E tests are engineered to execute completely offline with zero external network connectivity or cloud dependencies:

```
+---------------------------------------------------------------------------------------+
|                                  E2E Test Runner                                      |
|                               (go test -v ./test/e2e/...)                             |
+---------------------------------------------------------------------------------------+
        |                                                                 |
        v                                                                 v
+-------------------------------+                       +-------------------------------+
|    In-Memory gRPC Harness     |                       |   In-Memory WebSocket Harness |
|      (grpc/test/bufconn)      |                       |     (net/http/httptest)       |
+-------------------------------+                       +-------------------------------+
| - In-memory pipe (1 MB buffer)|                       | - Local ephemeral TCP listener|
| - Zero socket port contention |                       | - Real gorilla/websocket Conn |
| - Native gRPC Client & Server |                       | - Real Hub pub/sub broadcast  |
| - Full protobuf serialization |                       | - Read/Write pump validation  |
+-------------------------------+                       +-------------------------------+
        \                                                               /
         \                                                             /
          v                                                           v
+---------------------------------------------------------------------------------------+
|                         Mock WeatherFeedClient & Hub Core                             |
|  - Pre-seeded with authentic raw NOAA AWC bulletins (METAR, TAF, Winds Aloft, SIGMET)  |
|  - Synchronous cache injection via SetMockFeed(stationID, raw)                        |
|  - Zero HTTP external calls to aviationweather.gov                                    |
+---------------------------------------------------------------------------------------+
```

### 2.1 Technology Stack & Test Libraries
- **gRPC In-Memory Transport**: `google.golang.org/grpc/test/bufconn` provides an in-memory `net.Listener` allowing full client-server gRPC interaction without opening external firewall ports or binding to host TCP interfaces.
- **WebSocket Gateway Transport**: `net/http/httptest.NewServer` coupled with `github.com/gorilla/websocket` tests the complete RFC 6455 handshake, ping/pong heartbeats, frame streaming, and client disconnections.
- **Embedded Fixtures**: Authentic NOAA Aviation Weather Center bulletins covering standard, marginal, instrument, high-altitude, and severe storm environments.

---

## 3. Systematic 4-Tier Test Matrix

### 3.1 Tier 1: Feature Coverage (>= 5 Tests Per Feature)

| # | Feature Under Test | Test ID | Description | Input / Condition | Expected Output / Invariant |
|---|--------------------|---------|-------------|-------------------|-----------------------------|
| 1 | METAR Parsing | `T1_METAR_01` | Nominal VFR observation | `KAPA 050550Z 18010KT 10SM SCT050 18/04 A2995` | `Wind: 180@10kt`, `Vis: 10SM`, `Ceil: 25000ft (None)`, `Cat: VFR` |
| 2 | METAR Parsing | `T1_METAR_02` | Low IFR with ceiling & mist | `KASE 050550Z 34005KT 2SM OVC009 00/M01 A2985` | `Wind: 340@5kt`, `Vis: 2SM`, `Ceil: 900ft AGL`, `Cat: IFR` |
| 3 | METAR Parsing | `T1_METAR_03` | Marginal VFR (MVFR) | `KCOS 050550Z 19008KT 4SM BKN025 10/02 A2992` | `Vis: 4SM`, `Ceil: 2500ft AGL`, `Cat: MVFR` |
| 4 | METAR Parsing | `T1_METAR_04` | LIFR dense fog & ceiling | `KBJC 050550Z 00000KT 1/2SM FG OVC003 02/02 A3002` | `Vis: 0.5SM`, `Ceil: 300ft AGL`, `Cat: LIFR` |
| 5 | METAR Parsing | `T1_METAR_05` | High wind with gusts | `KDEN 050550Z 28025G40KT 10SM BKN120 12/M04 A2990` | `Wind: 280@25kt G 40kt`, `Gust: 40kt`, `Cat: VFR` |
| 6 | TAF Horizon | `T1_TAF_01` | Base forecast parsing | `TAF KAPA ... 18010KT P6SM SKC` | Base wind `180@10kt`, vis `>6SM`, sky clear |
| 7 | TAF Horizon | `T1_TAF_02` | Rapid change group (`FM`) | `FM051400 20015G25KT 5SM HZ BKN040` | New conditions active from 1400Z: gust 25kt, ceil 4000ft |
| 8 | TAF Horizon | `T1_TAF_03` | Temporary deterioration (`TEMPO`) | `TEMPO 0518/0522 2SM TSRA BKN020CB` | Identifies thunderstorm & 2SM visibility during 18-22Z window |
| 9 | TAF Horizon | `T1_TAF_04` | Becoming group (`BECMG`) | `BECMG 0512/0514 30012KT 6SM NSW` | Gradual transition to 300@12kt and 6SM |
| 10 | TAF Horizon | `T1_TAF_05` | Probability group (`PROB30`) | `PROB30 0520/0524 1SM +TSRA BKN010CB` | Flags 30% risk of 1SM +TSRA below minimums |
| 11 | Winds Aloft | `T1_WIND_01` | Standard tropospheric layer | `DEN 1812+15` at 6,000 ft MSL | `Dir: 180°`, `Speed: 12kt`, `Temp: +15°C` |
| 12 | Winds Aloft | `T1_WIND_02` | Sub-zero altitude layer | `DEN 2625-10` at 18,000 ft MSL | `Dir: 260°`, `Speed: 25kt`, `Temp: -10°C` |
| 13 | Winds Aloft | `T1_WIND_03` | High-speed offset (+50 code) | `DEN 771558` at 39,000 ft MSL | `Dir: 270°`, `Speed: 115kt`, `Temp: -58°C` |
| 14 | Winds Aloft | `T1_WIND_04` | Calm wind code (`9900`) | `DEN 9900+05` at 9,000 ft MSL | `Dir: 000°`, `Speed: 0kt (calm)`, `Temp: +5°C` |
| 15 | Winds Aloft | `T1_WIND_05` | Vector interpolation (u/v) | Interpolate between 6k (180@10) and 12k (270@20) at 9k ft | Returns smooth u/v vector: ~225° @ 14kt without angle wrap bug |
| 16 | SIGMET Hazard | `T1_SIG_01` | Convective SIGMET (`WST`) | Active line of severe thunderstorms | Parsed as `CONVECTIVE_SIGMET`, severity `EXTREME`, active `true` |
| 17 | SIGMET Hazard | `T1_SIG_02` | Severe Icing (`SEV ICE`) | Severe icing layer FL100-FL180 | Parsed as `SEVERE_ICING`, severity `SEVERE`, active `true` |
| 18 | SIGMET Hazard | `T1_SIG_03` | Severe Turbulence (`SEV TURB`) | Severe CAT FL150-FL250 | Parsed as `SEVERE_TURBULENCE`, severity `SEVERE`, active `true` |
| 19 | SIGMET Hazard | `T1_SIG_04` | Expired SIGMET handling | `ValidUntil` is in the past | Advisory parsed but `active == false` (no dispatch penalty) |
| 20 | SIGMET Hazard | `T1_SIG_05` | Spatial point-in-polygon check | Point inside bounding polygon vertices | Returns `intersects == true` |
| 21 | Part 135.L | `T1_REG_01` | Non-mtn day local | KAPA: Ceil 600ft, Vis 2SM, NVG equipped | Compliant (min req 500ft / 2SM), Verdict `GO` |
| 22 | Part 135.L | `T1_REG_02` | Non-mtn day local (standard) | KAPA: Ceil 600ft, Vis 2SM, Unaided | Non-compliant (min req 800ft / 2SM), Verdict `NO-GO` |
| 23 | Part 135.L | `T1_REG_03` | Non-mtn night local (unaided) | KAPA: Ceil 1000ft, Vis 3SM | Compliant (min req 1000ft / 3SM), Verdict `GO` |
| 24 | Part 135.L | `T1_REG_04` | Mtn night cross-country (NVG) | KLXV: Ceil 1000ft, Vis 3SM, NVG equipped | Compliant (min req 1000ft / 3SM), Verdict `GO` |
| 25 | Part 135.L | `T1_REG_05` | Mtn night cross-country (unaided) | KLXV: Ceil 1200ft, Vis 4SM, Unaided | Non-compliant (min req 1500ft / 5SM), Verdict `NO-GO` |
| 26 | gRPC Service | `T1_GRPC_01` | `GetStationWeather` valid | Request station `KAPA` | Returns `METARResponse` with 180@10kt, VFR |
| 27 | gRPC Service | `T1_GRPC_02` | `GetStationWeather` empty station | Request station `""` | Returns gRPC status code `InvalidArgument` |
| 28 | gRPC Service | `T1_GRPC_03` | `EvaluateFlightMinimums` compliant | KAPA day local with NVG | Returns `is_compliant == true`, verdict `GO` |
| 29 | gRPC Service | `T1_GRPC_04` | `EvaluateFlightMinimums` below mins | KASE 900ft mtn night unaided | Returns `is_compliant == false`, verdict `NO-GO` |
| 30 | gRPC Service | `T1_GRPC_05` | `StreamCorridorWeather` streaming | Corridor: `["KAPA", "KLXV"]`, interval 1s | Receives streamed updates via in-memory `bufconn` |
| 31 | WebSocket | `T1_WS_01` | Connection handshake & upgrade | Dial `ws://127.0.0.1:<port>/ws` | Connection upgraded to 101 Switching Protocols cleanly |
| 32 | WebSocket | `T1_WS_02` | `METAR_UPDATE` event broadcast | Hub broadcasts KDEN METAR | Client receives JSON `{"type": "METAR_UPDATE", ...}` |
| 33 | WebSocket | `T1_WS_03` | `MINIMUMS_EVAL` event broadcast | Hub broadcasts evaluation report | Client receives JSON `{"type": "MINIMUMS_EVAL", ...}` |
| 34 | WebSocket | `T1_WS_04` | Ping/Pong RFC 6455 heartbeat | Server sends ping frame | Client pong handler responds, connection stays healthy |
| 35 | WebSocket | `T1_WS_05` | Client subscription message | Client writes `SUBSCRIBE_STATIONS` | Server unmarshals client payload without error |

---

### 3.2 Tier 2: Boundary & Corner Cases

| # | Corner Case Category | Test ID | Input / Condition | Authoritative Spec / Expected Behavior |
|---|----------------------|---------|-------------------|-----------------------------------------|
| 1 | Compound Visibility | `T2_CORNER_01` | `1 1/2SM` or `2 1/2SM` in METAR | Correctly tokenized as single visibility: `1.5` SM and `2.5` SM (not truncated to `0.5` SM). |
| 2 | Minimum Visibility Prefix | `T2_CORNER_02` | `M1/4SM` in METAR | "M" prefix denotes "less than". Parsed as `0.25` SM with `IsLessThan = true`. Fails all FAA minimums. |
| 3 | Maximum Visibility Prefix | `T2_CORNER_03` | `P6SM` in METAR | "P" prefix denotes "greater than". Parsed as `6.0` SM with `IsGreaterThan = true`. |
| 4 | Clear Sky Variations | `T2_CORNER_04` | `CLR`, `SKC`, `CAVOK`, `NSC` | No ceiling layer below sensor limit. Evaluated as `HasCeiling = false`, ceiling = `+Inf` (passes all ceiling minimums). |
| 5 | Vertical Obscuration Ceiling | `T2_CORNER_05` | `VV002` (Vertical Visibility 200 ft) | Indefinite ceiling due to fog/surface obscuration. Constitutes a hard ceiling of `200` ft AGL under FAR 1.1. |
| 6 | Variable Wind Light | `T2_CORNER_06` | `VRB03KT` | Variable direction wind under 6 kts. Parsed as `IsVariable = true`, speed `3.0` kts, direction default 0. |
| 7 | Variable Wind Direction Shift | `T2_CORNER_07` | `18015KT 140V220` | Wind varying >= 60°. Max crosswind calculated against both sector extremes (140° and 220°). |
| 8 | Winds Aloft Speed Offset | `T2_CORNER_08` | Code `751548` at 30,000 ft | Direction code > 50 indicates speed >= 100kt. Dir: `(75-50)*10 = 250°`, Speed: `15 + 100 = 115kt`, Temp: `-48°C`. |
| 9 | Winds Aloft Calm Code | `T2_CORNER_09` | Code `9900` or `9900+00` | Standard NWS calm indicator. Decoded as `Speed = 0.0 kts`, `Direction = 000°`. |
| 10 | Sub-Zero Negative Temps | `T2_CORNER_10` | `M04/M12` in METAR | Prefix "M" indicates negative Celsius: Temp = `-4.0°C`, Dewpoint = `-12.0°C`. |
| 11 | High Density Altitude | `T2_CORNER_11` | Leadville KLXV (elev 9,927 ft MSL, temp 28°C, alt 29.70) | Density altitude exceeds `13,500` ft MSL. Pressure altitude calculated accurately. |
| 12 | Corrupted Sensor Feeds | `T2_CORNER_12` | `/////KT 10SM SCT025 M//M// A2992` | Unmeasured sensor slashes parsed gracefully without panic or unhandled exceptions. |

---

### 3.3 Tier 3: Cross-Feature Combinations

| # | Cross-Feature Interaction | Test ID | Setup & Input | Expected System Behavior |
|---|---------------------------|---------|---------------|--------------------------|
| 1 | Corridor Weather + Convective SIGMET | `T3_COMBO_01` | METAR reports 10SM CLR, but active Convective SIGMET intersects corridor | SIGMET takes precedence over surface METAR; immediate `DISQUALIFIED` verdict with limiting factor `ACTIVE_CONVECTIVE_SIGMET`. |
| 2 | Severe Icing Hazard | `T3_COMBO_02` | Severe icing advisory active along flight route | Disqualifies dispatch under FAA AC 91-74B; verdict `DISQUALIFIED` (`SEVERE_ICING_HAZARD`). |
| 3 | IFR Transition under § 135.611 | `T3_COMBO_03` | Ceiling is 400 ft / 1 SM (violating § 135.609), aircraft is IFR certified, alternate airport available, onboard radar active | Engine classifies verdict as `IFR_REQUIRED` with citation to 14 CFR § 135.611 instead of outright rejection. |
| 4 | IFR Disqualification | `T3_COMBO_04` | Ceiling is 400 ft, aircraft IFR certified, but NO alternate airport or radar inoperative | Fails § 135.611 prerequisites; verdict falls back to `BELOW_MINIMUMS` (`NO-GO`). |
| 5 | Multi-Station Corridor Worst-Case Aggregation | `T3_COMBO_05` | Corridor: KAPA (10SM CLR), KDEN (8SM BKN050), KLXV (10SM SCT080), KASE (2SM OVC009) | Engine evaluates the union: worst ceiling is `900 ft` (KASE) and worst visibility is `2.0 SM` (KASE). Dictates corridor status. |
| 6 | gRPC & WebSocket Dual-Serving Synchronization | `T3_COMBO_06` | Ingest new METAR or minimums evaluation via gRPC | WebSocket broadcast hub automatically dispatches corresponding event to all connected web clients in real time. |

---

### 3.4 Tier 4: Real-World Application Scenarios

#### Scenario 1: Denver Metro Nominal HEMS (`T4_SCENARIO_01`)
- **Route**: Centennial Airport (`KAPA`) -> Denver Health Medical Center (`KDEN`).
- **Context**: Non-mountainous, Day, Local Flying Area (< 50 NM), unaided pilot.
- **Conditions**: `KAPA 050550Z 18010KT 10SM SCT050 18/04 A2995` / `KDEN 050550Z 17012KT 10SM BKN060 19/04 A2994`.
- **Statutory Rule**: 14 CFR § 135.609 Table 1 [Non-mountainous local - Day (standard)] -> Min Ceiling 800 ft, Min Vis 2 SM.
- **Observed**: Ceiling 6,000 ft AGL, Visibility 10.0 SM.
- **Verdict**: `APPROVED` / `GO` (Ceiling margin +5,200 ft, Vis margin +8.0 SM).

#### Scenario 2: Leadville High-Density Altitude Mountain Rescue (`T4_SCENARIO_02`)
- **Route**: Centennial Airport (`KAPA`) -> Leadville Lake County Airport (`KLXV`).
- **Context**: Mountainous, Day, Cross-Country (> 50 NM into Front Range / Sawatch Range), NVG/HTAWS equipped.
- **Conditions**: `KLXV 050550Z 26012KT 10SM SCT080 28/02 A2970` (Elevation 9,927 ft MSL, ambient temp 28°C).
- **Statutory Rule**: 14 CFR § 135.609 Table 1 [Mountainous cross-country - Day] -> Min Ceiling 1,000 ft, Min Vis 3 SM.
- **Physical Evaluation**: Density Altitude = `13,634 ft MSL`. Severe OGE hover ceiling deficit identified.
- **Verdict**: Weather compliant for transit (`GO`), but flags extreme density altitude operational caution.

#### Scenario 3: Aspen Mountain Snow Blizzard Diversion (`T4_SCENARIO_03`)
- **Route**: Aspen Pitkin County (`KASE`) -> St. Anthony Hospital (`KAPA`).
- **Context**: Mountainous, Night, Cross-Country, NVG/HTAWS equipped.
- **Conditions**: `KASE 050550Z 34005KT 2SM -SN OVC009 00/M01 A2985`.
- **Statutory Rule**: 14 CFR § 135.609 Table 1 [Mountainous cross-country - Night (with NVG/HTAWS)] -> Min Ceiling 1,000 ft, Min Vis 3 SM.
- **Observed**: Ceiling 900 ft AGL (Deficit: -100 ft), Visibility 2.0 SM (Deficit: -1.0 SM).
- **Verdict**: `BELOW_MINIMUMS` / `NO-GO` (Limiting factor: `FAR_135_615_CEILING_BELOW_MINIMUMS`).
- **Diversion Engine**: Evaluate en-route alternate `KGJT` (Grand Junction: 10SM CLR). Re-routes safely under § 135.611 / alternate procedure.

---

## 4. Test Execution & Verification

### 4.1 Running the Complete E2E Suite
From PowerShell with the configured Go toolchain:

```powershell
$env:PATH = "C:\Users\Nick Napoli\go\bin;C:\Users\Nick Napoli\go_sdk\go\bin;" + $env:PATH
cd "d:\SN850X\Home\Documents\Projects\TraumaTeam\services\weather_ingest"
go test -v -count=1 ./test/e2e/...
```

### 4.2 Test Pass Criteria
- **100% Pass Rate**: Every test across Tiers 1-4 passes without assertion failures or panics.
- **Zero External Network**: Wi-Fi/Ethernet disconnection has zero effect on test outcome.
- **Fast Execution**: Entire E2E suite completes in < 5 seconds due to in-memory `bufconn` and `httptest` design.
- **Deterministic**: No test order dependence, shared state leaks, or timing flakiness.

---

## 5. Escalation & Quality Assurance Record

The E2E Test Architect identified the following behavioral nuances during specification verification:
1. **Regulatory Section Disambiguation**: The repository's legacy Python modules cite "14 CFR § 135.615 Table 1". Official eCFR documentation establishes that Table 1 is codified under **14 CFR § 135.609**, while § 135.615 governs VFR obstacle clearance (300 ft Day / 500 ft Night). The Go test harness tests against both the statutory citations and the operational thresholds.
2. **Compound Fractional Visibility**: Raw METAR tokens such as `1 1/2SM` require specialized lookahead tokenization to avoid discarding the leading whole integer.
3. **Slow Consumer Protection**: WebSocket clients unable to drain channel queues within backpressure deadlines are safely dropped to protect high-priority telemetry delivery to other HEMS dispatch consoles.
