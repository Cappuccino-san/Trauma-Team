# Contributing to TraumaTeam

Thank you for your interest in contributing to **TraumaTeam**! TraumaTeam is an aviation flight operations engineering and dispatch decision engine purpose-built for Helicopter Emergency Medical Services (HEMS) operating under **FAA 14 CFR Part 135 Subpart L** (§§ 135.601–135.621), Part 91, and Part 135 Subpart D.

Because this project models safety-critical flight dynamics and regulatory compliance, we maintain rigorous engineering standards across our polyglot architecture:
- **Rust** (`crates/traumateam_rfm`): Aerodynamics, RFM multi-dimensional interpolation, kinematics, C-ABI FFI.
- **Go** (`services/weather_ingest`): NOAA AWC decoders, gRPC `:50051`, WebSocket hub `:8080`, CGO bridge.
- **Python** (`traumateam/` & `tests/`): Reference engine, atmospheric physics, wind triangles, Hypothesis property tests, CLI tools.
- **TypeScript / Svelte 5** (`frontend/`): Dispatcher avionics console HUD.

---

## Code of Conduct

All contributors are expected to uphold the [Contributor Covenant v2.1](CODE_OF_CONDUCT.md). Please report any unacceptable behavior to [security@traumateam.internal].

---

## Local Development Setup

### Prerequisites

Ensure the following toolchains are installed on your system:
- **Python**: >= 3.10 (managed with [`uv`](https://github.com/astral-sh/uv))
- **Rust**: >= 1.85 (with `cargo` and `clippy`)
- **Go**: >= 1.24 (with `cgo` and GCC/Clang)
- **Node.js**: >= 20.x (with `npm`)
- **Docker & Compose**: (optional, for containerized deployments)

### Quick Environment Setup

Clone the repository and inspect the toolchains:

```bash
git clone https://github.com/Cappuccino-san/Trauma-Team.git
cd Trauma-Team

# Install Python dependencies in virtual environment via uv
uv sync --extra dev

# Install Frontend dependencies
cd frontend && npm install && cd ..
```

---

## Building and Testing

TraumaTeam provides a root `Makefile` that orchestrates all four language stacks.

### 1-Command Polyglot Build & Test

```bash
# Build all components (Rust cdylib, Go gateway binary, Frontend assets)
make build

# Run all 4 test suites (Python, Rust, Go, Svelte) - 100% offline and deterministic
make test
```

### Running Specific Test Suites

```bash
# Python pytest & Hypothesis property tests
make test-python
# or directly:
uv run --extra dev pytest -v

# Rust unit, benchmark, and FFI tests
make test-rust
# or directly:
cargo test --manifest-path crates/traumateam_rfm/Cargo.toml

# Go NOAA decoding, gRPC, and E2E WebSocket tests
make test-go
# or directly:
cd services/weather_ingest && go test -v -count=1 ./...

# Svelte 5 frontend component & store tests
make test-frontend
# or directly:
cd frontend && npm test
```

### Running Linters & Type Checks

```bash
make lint
```
This executes:
1. `cargo clippy --manifest-path crates/traumateam_rfm/Cargo.toml -- -D warnings`
2. `go vet ./...` in `services/weather_ingest`
3. `svelte-check --tsconfig ./tsconfig.json && tsc --noEmit` in `frontend`

---

## Interactive Demo Execution

Before submitting changes, ensure the scenario demonstrators run cleanly:

```bash
# Run multi-scenario dispatch demonstration
make demo
# or:
uv run traumateam-demo
# or:
uv run python demo_dispatch.py

# Test Python CLI entry point
uv run traumateam --help
uv run traumateam airframe --model H145
uv run traumateam regulations --environment mountainous --flight-phase night --unaided
```

To test the standalone zero-dependency web interface:
```bash
# Open directly in any modern browser:
xdg-open traumateam_console.html   # On Linux
open traumateam_console.html       # On macOS
```

---

## Code Standards & Style Guidelines

### 1. General Principles
- **Aviation Rigor**: No arbitrary fudge factors or approximations without referencing OEM RFMs or FAA regulations.
- **Safety Critical Invariants**: Never bypass ceiling deficits, fuel reserves (FAR 135.209 / 135.223), or lateral/longitudinal CG limits.
- **Air-Gapped & Offline Testing**: All test suites must pass 100% offline without live internet access. Mock all external NOAA feeds using deterministic test fixtures.

### 2. Rust (`crates/traumateam_rfm`)
- Zero warnings: code must pass `cargo clippy -- -D warnings`.
- Format code with `cargo fmt`.
- Protect all unsafe C-ABI functions with pointer validation and error return codes.

### 3. Go (`services/weather_ingest`)
- Idiomatic Go formatted via `gofmt` and checked with `go vet`.
- Manage goroutine lifetimes cleanly with context cancellation.

### 4. Python (`traumateam/`)
- Type annotations required on all public methods and dataclasses.
- Verify adherence with `pyright` or `mypy`.
- Maintain property-based test invariants using `hypothesis`.

### 5. Frontend (`frontend/`)
- Strict TypeScript with zero errors under `svelte-check`.
- Responsive, minimalist avionics styling using Tailwind CSS.

---

## Pull Request Guidelines

1. **Branch Naming**:
   - `feature/<description>` for new capabilities
   - `fix/<description>` for bug or performance fixes
   - `chore/<description>` for packaging, infrastructure, or docs
2. **Commit Messages**:
   - Use conventional commit format: `feat:`, `fix:`, `chore:`, `docs:`, `test:`, `refactor:`.
   - Explain *why* a calculation or regulation was adjusted.
3. **Verification**:
   - Every PR must verify that `make test` and `make lint` pass with 100% success.
   - Fill out `.github/pull_request_template.md` completely.
