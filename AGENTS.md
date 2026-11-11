# TraumaTeam Workspace Rules & Operational Guidelines

## 1. Container Healthcheck Loopback Precision
- When authoring `HEALTHCHECK` instructions in Dockerfiles using Alpine Linux or minimal container images, prefer explicit IPv4 loopback (`127.0.0.1`) rather than `localhost` (e.g., `wget -qO- http://127.0.0.1/ || exit 1` or `curl -f http://127.0.0.1:<port>/ || exit 1`).
- This prevents `Connection refused` flakes caused by `localhost` resolving to IPv6 `::1` while the daemon is bound exclusively to IPv4 `0.0.0.0`.

## 2. Nginx WebSocket Reverse Proxy Configuration
- Always include the HTTP/1.1 upgrade map when proxying WebSocket endpoints:
  ```nginx
  map $http_upgrade $connection_upgrade {
      default upgrade;
      ''      close;
  }
  ```
- Ensure proxy locations pass:
  ```nginx
  proxy_http_version 1.1;
  proxy_set_header Upgrade $http_upgrade;
  proxy_set_header Connection $connection_upgrade;
  proxy_set_header Host $host;
  ```

## 3. Dynamic Native Library Linking (cgo / cdylib)
- When distributing Rust cdylib binaries (`.so`) linked into Go binaries across multi-stage Docker builds, always embed linker rpaths via cgo:
  ```go
  // #cgo LDFLAGS: -L/usr/local/lib -ltraumateam_rfm -Wl,-rpath,/usr/local/lib -Wl,-rpath,/usr/lib
  ```
- Pair this with setting `ENV LD_LIBRARY_PATH=/usr/local/lib:/usr/lib:${LD_LIBRARY_PATH}` and invoking `ldconfig` in the final runtime stage.

