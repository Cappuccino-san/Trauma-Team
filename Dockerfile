# ==============================================================================
# TraumaTeam: Deterministic HEMS Mission Feasibility & Dispatch Engine
# Multi-stage Containerization Pipeline (Rust -> Go -> Svelte -> Nginx)
# ==============================================================================

# ------------------------------------------------------------------------------
# Stage 1: Rust Physics & RFM Core (cdylib)
# ------------------------------------------------------------------------------
FROM rust:1.85-bookworm AS rust-builder
WORKDIR /usr/src/traumateam

COPY crates/traumateam_rfm ./crates/traumateam_rfm

WORKDIR /usr/src/traumateam/crates/traumateam_rfm
RUN cargo build --release

# ------------------------------------------------------------------------------
# Stage 2: Go Telemetry & Compliance Gateway (cgo)
# ------------------------------------------------------------------------------
FROM golang:1.24-bookworm AS go-builder
WORKDIR /usr/src/traumateam

RUN apt-get update && apt-get install -y --no-install-recommends build-essential && rm -rf /var/lib/apt/lists/*

# Copy compiled Rust cdylib and symbols for cgo linker
COPY --from=rust-builder /usr/src/traumateam/crates/traumateam_rfm/target/release /usr/src/traumateam/crates/traumateam_rfm/target/release
COPY crates/traumateam_rfm/src/ffi.rs /usr/src/traumateam/crates/traumateam_rfm/src/ffi.rs

# Copy Go weather_ingest source code
COPY services/weather_ingest ./services/weather_ingest

WORKDIR /usr/src/traumateam/services/weather_ingest
RUN go mod download
ENV CGO_ENABLED=1
ENV GOOS=linux
RUN go build -ldflags="-s -w" -o /usr/local/bin/gateway-service main.go

# ------------------------------------------------------------------------------
# Stage 3: Svelte Edge UI Builder (Vite static build)
# ------------------------------------------------------------------------------
FROM node:22-alpine AS frontend-builder
WORKDIR /usr/src/traumateam/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# ------------------------------------------------------------------------------
# Stage 4: Production Runtime - Go Telemetry Gateway
# ------------------------------------------------------------------------------
FROM debian:bookworm-slim AS runtime-gateway
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install Rust shared library and update ld cache
COPY --from=rust-builder /usr/src/traumateam/crates/traumateam_rfm/target/release/libtraumateam_rfm.so /usr/local/lib/libtraumateam_rfm.so
COPY --from=rust-builder /usr/src/traumateam/crates/traumateam_rfm/target/release/libtraumateam_rfm.so /usr/lib/libtraumateam_rfm.so
ENV LD_LIBRARY_PATH=/usr/local/lib:/usr/lib:${LD_LIBRARY_PATH}
RUN ldconfig

# Install Go Gateway executable
COPY --from=go-builder /usr/local/bin/gateway-service /usr/local/bin/gateway-service

EXPOSE 8080 50051

HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/gateway-service"]

# ------------------------------------------------------------------------------
# Stage 5: Production Runtime - Svelte Edge UI (Nginx)
# ------------------------------------------------------------------------------
FROM nginx:alpine AS runtime-ui
COPY --from=frontend-builder /usr/src/traumateam/frontend/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

HEALTHCHECK --interval=10s --timeout=5s --start-period=3s --retries=3 \
  CMD wget -qO- http://127.0.0.1/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
