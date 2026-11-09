# ==============================================================================
# TraumaTeam: Polyglot HEMS Mission Feasibility & Dispatch Engine
# Unified Root Makefile for Python, Rust, Go, and TypeScript/Svelte stacks
# ==============================================================================

SHELL := /usr/bin/env bash

.PHONY: all build test test-python test-rust test-go test-frontend demo lint clean docker-up help

all: build test

help:
	@echo "TraumaTeam Master Build & Test Automation"
	@echo "=========================================="
	@echo "  make build          Build all language targets (Rust cdylib, Go binary, Frontend)"
	@echo "  make test           Run all 4 test suites (Python, Rust, Go, Svelte)"
	@echo "  make test-python    Run Python pytest and Hypothesis suites"
	@echo "  make test-rust      Run Rust cargo test suite"
	@echo "  make test-go        Run Go test suite (100% offline E2E)"
	@echo "  make test-frontend  Run Svelte 5 frontend vitest suite"
	@echo "  make demo           Execute TraumaTeam scenario demonstration CLI"
	@echo "  make lint           Run code linters across all components"
	@echo "  make clean          Clean transient build artifacts and caches"
	@echo "  make docker-up      Build and run docker compose services"

build: build-rust build-go build-frontend

build-rust:
	@echo "[+] Building Rust RFM core (crates/traumateam_rfm)..."
	cargo build --manifest-path crates/traumateam_rfm/Cargo.toml --release

build-go:
	@echo "[+] Building Go telemetry & compliance gateway (services/weather_ingest)..."
	cd services/weather_ingest && go build -ldflags="-s -w" -o gateway-service main.go

build-frontend:
	@echo "[+] Building Svelte 5 dispatcher HUD (frontend)..."
	cd frontend && npm run build

test: test-python test-rust test-go test-frontend
	@echo "================================================================="
	@echo " ALL 4 POLYGLOT TEST SUITES PASSED CLEANLY (Rust, Go, Py, Svelte)"
	@echo "================================================================="

test-python:
	@echo "[+] Running Python verification suite (pytest)..."
	uv run --extra dev pytest

test-rust:
	@echo "[+] Running Rust RFM & FFI kinematics suite (cargo test)..."
	cargo test --manifest-path crates/traumateam_rfm/Cargo.toml

test-go:
	@echo "[+] Running Go weather ingest & E2E compliance suite (go test)..."
	cd services/weather_ingest && go test -v -count=1 ./...

test-frontend:
	@echo "[+] Running Svelte frontend unit & component tests (vitest)..."
	cd frontend && npm test

demo:
	@echo "[+] Executing TraumaTeam HEMS Dispatch Demonstration..."
	uv run traumateam-demo

lint:
	@echo "[+] Linting Rust..."
	cargo clippy --manifest-path crates/traumateam_rfm/Cargo.toml -- -D warnings
	@echo "[+] Linting Go..."
	cd services/weather_ingest && go vet ./...
	@echo "[+] Typechecking Frontend..."
	cd frontend && npm run check

clean:
	@echo "[+] Cleaning temporary artifacts..."
	rm -rf traumateam.egg-info
	rm -rf .pytest_cache .hypothesis
	rm -rf services/weather_ingest/gateway-service
	cargo clean --manifest-path crates/traumateam_rfm/Cargo.toml
	rm -rf frontend/dist
