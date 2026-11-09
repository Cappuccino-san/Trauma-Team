# Project: Trauma-Team Repository Optimization, Packaging, and Open Source Delivery

## Architecture
Trauma-Team is a safety-critical Helicopter Emergency Medical Services (HEMS) mission feasibility and dispatch decision engine under FAA 14 CFR Part 135 Subpart L.
- **Physics Core**: Rust (`crates/traumateam_rfm`) compiling to `cdylib` and `rlib`.
- **Weather & Telemetry Gateway**: Go (`services/weather_ingest`) with gRPC `:50051`, WebSocket `:8080`, and CGO FFI to Rust.
- **Reference & Verification Engine**: Python (`traumateam/`, `tests/`) with NumPy, SciPy, Pydantic, Hypothesis.
- **Dispatcher HUD UI**: Svelte 5 / TypeScript (`frontend/`) and standalone zero-dependency console (`traumateam_console.html`).
- **Containerization**: Multi-stage `Dockerfile` and `docker-compose.yml`.

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Dead Code & Artifact Cleanup | Remove legacy/redundant `services/weather_ingest/pkg/websocket`, duplicate main in `cmd/server`, root egg-info, clean temporary artifacts | M1 | Survey |
| 2 | Packaging & CLI Scripts | Modernize `pyproject.toml` with `[project.scripts]` (`traumateam = "traumateam.cli:main"`, `traumateam-demo = "demo_dispatch:main"`), `traumateam/__main__.py`, and typing marker `py.typed` | M2 | Survey |
| 3 | Root Multi-Stack Makefile | Top-level `Makefile` with targets (`all`, `build`, `test`, `demo`, `lint`, `clean`, `docker-up`) unifying the 4 language stacks | M2 | Survey |
| 4 | Rich Multi-Scenario Demo Runner | Upgrade `demo_dispatch.py` to an interactive/benchmark multi-scenario suite (Nominal Lowland GO, High DA OGE Deficit, Severe Convective SIGMET, Jet-Streak Fuel Exhaustion, Bell 429 Lateral CG Asymmetry) with formatted dispatch release tables | M3 | Survey |
| 5 | GitHub Community Governance | Create `LICENSE` (MIT), `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md` | M4 | Survey |
| 6 | GitHub Issue & PR Templates | Create `.github/ISSUE_TEMPLATE/bug_report.md`, `feature_request.md`, and `.github/pull_request_template.md` | M4 | Survey |
| 7 | GitHub Actions CI/CD Pipeline | Create `.github/workflows/ci.yml` matrix pipeline running automated tests across Rust, Go, Python (via `uv`), and Frontend | M4 | Survey |
| 8 | Polished README & Docs | Update `README.md` with status badges, quickstart guides (CLI, standalone web console, Docker), and cross-platform instructions; clean hardcoded Windows paths in docs | M4 | Survey |
| 9 | Git Branch & GitHub Pull Request | Cut feature branch `chore/oss-packaging-and-demo-polish`, commit all additions, push to GitHub remote, and execute `gh pr create` with rich release notes | M5 | Survey |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| 1 | Cleanup & Code Optimization | Clean obsolete packages, duplicate main entries, and transient build artifacts | none | DONE |
| 2 | Packaging & Build Automation | `pyproject.toml` CLI entry points, `__main__.py`, `py.typed`, root `Makefile` | M1 | DONE |
| 3 | Demo Polish & Verification | Multi-scenario aviation benchmark dispatch runner, verify standalone console | M2 | DONE |
| 4 | GitHub Infrastructure & Docs | GitHub CI matrix workflow, community health files, issue/PR templates, README | M3 | DONE |
| 5 | Branching, Audit & PR Creation | Feature branch creation, full test suite audit, push to origin, `gh pr create` | M4 | IN_PROGRESS |

## Interface Contracts
### CLI & Entry Points
- `uv run traumateam --help` or `python -m traumateam`: Python package CLI entry point.
- `uv run python demo_dispatch.py` or `uv run traumateam-demo`: Multi-scenario dispatch benchmark.
- Root `make test`: Executes all 4 test suites (Rust, Go, Python, Svelte).
- Root `make demo`: Executes CLI demo runner.
