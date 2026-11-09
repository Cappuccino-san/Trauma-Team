---
name: Bug Report
about: Create a report to help us fix a defect or calculation error
title: "[BUG]: "
labels: ["bug"]
assignees: ""
---

### Component Affected
- [ ] Physics & Kinematics Engine (Rust `crates/traumateam_rfm`)
- [ ] Weather & Telemetry Gateway (Go `services/weather_ingest`)
- [ ] Python Reference Engine (`traumateam/`)
- [ ] Dispatcher Avionics Console (Svelte `frontend/`)
- [ ] Standalone Avionics Console (`traumateam_console.html`)
- [ ] Packaging / CLI / Docker

### Describe the Bug
A clear and concise description of what the bug is. If this involves an erroneous dispatch verdict (e.g. false `GO` when minimums are violated, or incorrect fuel burn), please state the expected vs observed outcome.

### Aircraft & Mission Parameters
- **Airframe**: [e.g. Airbus H145 / Bell 429]
- **Gross Weight / Payload**: [e.g. 3,450 kg, 80 kg patient]
- **Weather / Corridor**: [e.g. METAR KAPA 18010KT 2SM OVC007, Temperature 25C, Altimeter 29.92]
- **Operating Environment**: [e.g. Mountainous / Night / Local]

### Steps to Reproduce
1. Execute command '...' or trigger scenario '...'
2. Observe output '...'
3. See error or discrepancy '...'

### Expected Behavior
A clear and concise description of what you expected to happen under FAA 14 CFR Part 135 regulations or OEM RFM tabular performance.

### Environment & Toolchain Details
- **OS**: [e.g. Linux Ubuntu 24.04, macOS 14.5, Windows 11 WSL2]
- **Python Version**: [e.g. 3.11 with uv]
- **Rust Version**: [e.g. 1.85+]
- **Go Version**: [e.g. 1.24+]
- **Node Version**: [e.g. 20.x]

### Additional Context
Add any other context, terminal logs, or screenshots about the problem here.
