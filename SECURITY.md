# Security Policy & Vulnerability Reporting

## Aviation Safety-Critical Systems Notice

TraumaTeam is a mission-feasibility and dispatch decision engine designed to operate in accordance with **FAA 14 CFR Part 135 Subpart L** (§§ 135.601–135.621), Part 91, and Part 135 Subpart D.

Because this software evaluates flight safety parameters, aerodynamic weight and balance envelopes, hover ceiling deficits, and regulatory legal minimums:
- Correctness, determinism, and mathematical precision are paramount.
- Any bug or vulnerability that could lead to an incorrect `GO` recommendation, false fuel reserve calculation, undetected center-of-gravity exceedance, or misparsed meteorological warning (e.g. convective SIGMET omission) is treated as a **Critical Safety Defect**.

---

## Supported Versions

Only active release lines receive security updates and safety patches:

| Version | Supported          | Severity Classes Accepted |
| ------- | ------------------ | ------------------------- |
| 1.0.x   | :white_check_mark: | Critical, High, Medium    |
| < 1.0   | :x:                | None (End of Life)        |

---

## Reporting a Security Vulnerability or Safety-Critical Defect

If you discover a security vulnerability or algorithmic flight-safety flaw, please **do not open a public GitHub issue**. Public disclosure before mitigation can endanger operations relying on deterministic dispatch models.

### Coordinated Disclosure Procedure

1. **Email Notification**: Send a detailed advisory to:
   - Email: `security@traumateam.internal` (or repository security contact)
   - Subject: `[SAFETY VULNERABILITY] TraumaTeam - <Component/Module> - <Brief Summary>`

2. **Required Information in Report**:
   - **Component Affected**: (e.g., `crates/traumateam_rfm`, `services/weather_ingest`, `traumateam/regulations`, `frontend`)
   - **Vulnerability / Flaw Type**:
     - *Flight Safety Logic*: Erroneous dispatch verdict, incorrect density altitude formula, bypass of FAR 135.615 minimums, incorrect CG limits.
     - *Memory Safety / Crash*: CGO/FFI memory corruption, buffer overflow, uncontrolled panic/segfault.
     - *Ingestion / Denial of Service*: Malformed METAR/TAF/SIGMET parser loop or network crash.
     - *Authentication / Transport*: WebSocket session hijacking, CORS misconfiguration.
   - **Reproduction Steps**: Complete scenario configuration (aircraft model, payload weights, METAR strings, waypoints) or minimal test case demonstrating the incorrect output.
   - **Impact Assessment**: Operational consequences under FAA Part 135 flight operations.
   - **Suggested Remediation**: If you have identified a fix or patch.

3. **Response Timelines**:
   - **Initial Acknowledgement**: Within 24 hours.
   - **Safety & Triage Assessment**: Within 48 hours.
   - **Remediation & Patch Release**: Within 7 business days for critical safety or security issues.

4. **Public Disclosure**:
   - Coordinated disclosure will occur following the release of an official patch or release advisory.
   - Credit will be given to the reporting researcher in our release notes and changelog (unless anonymity is requested).

---

## Secure Engineering Practices

- **Zero Tolerance for Warnings**: All Rust code builds with `#![deny(warnings)]` and Clippy lint enforcement.
- **Strict FFI Validation**: All C-ABI boundary functions (`crates/traumateam_rfm/src/ffi.rs`) perform strict null-pointer validation, bounds checking, and NaN/Infinity sanitization.
- **Air-Gapped Reliability**: All test suites must pass 100% offline without live internet access or unpinned external dependencies.
- **Property-Based Invariants**: Safety invariants are enforced continuously using Hypothesis in Python and adversarial fuzzing in Rust.
