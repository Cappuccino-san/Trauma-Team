## Summary of Changes

Provide an overview of the motivation, changes introduced, and how this PR enhances TraumaTeam.

### Type of Change
- [ ] :bug: Bug fix (non-breaking change fixing an issue or regulatory calculation)
- [ ] :sparkles: New feature (non-breaking change adding new airframe, feed decoder, or UI capability)
- [ ] :zap: Performance optimization (improving speed, memory usage, or numerical precision)
- [ ] :books: Documentation / Community standards update
- [ ] :hammer_and_wrench: Packaging / Build automation / CI/CD pipeline
- [ ] :shield: Safety-critical algorithm adjustment

---

## Safety & Regulatory Verification Checklist

As TraumaTeam models FAA 14 CFR Part 135 Subpart L compliance, check all applicable items:

- [ ] **FAA Part 135 Subpart L Compliance**: Weather minimums (§ 135.609 / § 135.615) are strictly maintained.
- [ ] **Fuel Reserves Compliance**: VFR (20-min / § 135.209) and IFR (§ 135.223) reserve calculations verified.
- [ ] **OEM RFM Integrity**: No arbitrary adjustments made to OEM Rotorcraft Flight Manual performance envelopes (Airbus H145 / Bell 429).
- [ ] **Offline Determinism**: All calculations and unit tests pass 100% offline without live internet access.
- [ ] **C-ABI / Memory Safety**: Any CGO/Rust FFI changes preserve pointer validation, non-null safety, and error code propagation.

---

## Polyglot Multi-Stack Verification Table

Run `make test` and `make lint` before submitting:

| Component | Test Command | Result |
| :--- | :--- | :--- |
| **Rust RFM Core** | `cargo test --manifest-path crates/traumateam_rfm/Cargo.toml` | [ ] Pass |
| **Rust Clippy** | `cargo clippy --manifest-path crates/traumateam_rfm/Cargo.toml -- -D warnings` | [ ] Pass |
| **Go Gateway** | `cd services/weather_ingest && go test -v -count=1 ./...` | [ ] Pass |
| **Go Vet** | `cd services/weather_ingest && go vet ./...` | [ ] Pass |
| **Python Pytest** | `uv run --extra dev pytest` | [ ] Pass |
| **Frontend Vitest** | `cd frontend && npm test` | [ ] Pass |
| **Frontend Check** | `cd frontend && npm run check` | [ ] Pass |

---

## Benchmark Scenario & Demo Verification

- [ ] Executed `uv run traumateam-demo` (or `make demo`) with all benchmark scenarios passing cleanly.
- [ ] Verified standalone web console `traumateam_console.html` loads correctly.
- [ ] Verified CLI entry points: `traumateam --help`.

---

## Related Issues / Tickets

Closes #
