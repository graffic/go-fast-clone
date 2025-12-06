# Fast.com Clone – Backend Design

## Overview

The backend is a Go HTTP service that:

- Serves the patched `original-webapp/` static files at `/`.
- Implements the HTTP APIs required by the fast.com web app for:
  - OCA directory discovery.
  - Speed test (download, upload, latency) endpoints.
  - Telemetry ingestion.

The design is based on `docs/backend-working-plan.md` and the current Go implementation under `cmd/api` and `internal/`.


## Process and Configuration

### Entry Point and Startup

- **Main binary**: `cmd/api/main.go`.
- Startup sequence:
  1. Configure logging via `setupLogging()` using zerolog.
  2. Load configuration via `internal/config.Load()`.
  3. Create HTTP server via `internal/server.New(cfg)`.
  4. Call `ListenAndServe()` and log fatal errors.

### Configuration

`internal/config/config.go` defines a `Config` struct and loads values from environment variables:

- `ListenAddr` (`LISTEN_ADDR`, default `":8080"`)
  - Address and port the HTTP server listens on.
- `StaticDir` (`STATIC_DIR`, default `"original-webapp"`)
  - Directory served as static files at `/`.
- `RandomDataFile` (`RANDOM_DATA_FILE`, default `"/dev/shm/random_data"`)
  - Path to the file (ideally on tmpfs) that stores pre-generated random bytes used as the source for download payloads.


## HTTP Server and Middleware

### Routing

`internal/server/server.go` defines the `Server` type, creates an `http.ServeMux`, and registers all routes:

- Static assets:
  - `"/"` → `http.FileServer(http.Dir(cfg.StaticDir))` (catch-all; registered last).
- OCA directory:
  - `"GET /netflix/speedtest/v2"` → `oca.HandleDirectory`.
- Speed test endpoints:
  - `"GET /speedtest/range/{range}"` → `speedHandler.HandleDownloadRange` (`SpeedTestHandler`).
  - `"POST /speedtest/range/{range}"` → `speedHandler.HandleUploadRange` (`SpeedTestHandler`).
  - `"GET /speedtest"` → `speedHandler.HandleDownload` (`SpeedTestHandler`).
  - `"POST /speedtest"` → `speedHandler.HandleUpload` (`SpeedTestHandler`).
- Telemetry:
  - `"POST /telemetry/cl2"` → `telemetry.HandleIngest`.

The Go 1.22+ pattern-based routing on `http.ServeMux` is used (e.g., `GET /path`, `{range}` wildcard).

### Middleware Stack

`Server.Handler()` wraps the mux with:

1. `LoggingMiddleware` – logs request/response metrics.


## OCA Directory Service

### Purpose

Implements the fast.com OCA directory endpoint (`/netflix/speedtest/v2`), which returns a list of speed test targets and client information.

### Data Structures

In `internal/oca/handler.go`:

- `Location { City, Country }`
- `Target { Name, URL, Location, ID, Label }`
- `ClientInfo { IP, ASN, ISP, Location }`
- `DirectoryResponse { Targets []Target, Client ClientInfo }`

These structures are JSON-annotated to match the expected response shape.

### Handler Behavior

Handler: `HandleDirectory(w http.ResponseWriter, r *http.Request)`.

- Currently implemented as a simple local configuration:
  - `baseURL := "/speedtest"`.
  - `Location` is hardcoded as `LocalCity`, `LC`.
  - `Client.IP` is derived from `r.RemoteAddr`.
  - `Client.ASN` is a dummy value (`"65535"`).
  - `Targets` contains a single entry:
    - `Name: baseURL`, `URL: baseURL`, `Location: location`.
- Response:
  - `Content-Type: application/json`.
  - JSON encoding of `DirectoryResponse`.

### Planned Evolution

Based on `backend-working-plan.md`, future iterations should:

- Use `cfg.BaseURL` to generate fully-qualified URLs like `https://host/speedtest/range/0-0`.
- Parse query parameters such as `https`, `token`, `urlCount`.
- Support multiple OCAs and more realistic `client` info.
- Optionally track OCA health and avoid unhealthy targets.


## Speed Test Endpoints

### Design Goals

- Provide realistic bandwidth testing by streaming large bodies for download and accepting large bodies for upload.
- Follow the semantics expected by the fast.com JS client, especially the `/speedtest/range/0-{N}` form.

### Shared State and Initialization

`internal/speed/handler.go` defines:

- `ReadBufferSize int64 = 30 * 1024 * 1024` (30 MiB).
- `MaxPayloadBytes int64 = 25 * 1024 * 1024`.
- `SpeedTestHandler` struct with `randomBufferFile` path.

`NewSpeedTestHandler(randomBufferFile string)` fills a `ReadBufferSize`-sized slice with random bytes using a time-seeded PRNG and writes it to the configured `randomBufferFile` (typically on tmpfs). This file is reused for download responses and allows the kernel to use `sendfile`/zero-copy I/O when serving speed test payloads.

### Performance Notes

- For best throughput and minimal latency, `RandomDataFile` should point to a tmpfs-backed path (e.g., `/dev/shm/random_data`) so that downloads hit memory instead of disk.
- Using `io.CopyN` from this file to the HTTP response allows Go’s `net/http` server and the OS kernel to use zero-copy paths (such as `sendfile`) where available, reducing CPU usage under load.

### Range-based Download

Route: `GET /speedtest/range/{range}` → `SpeedTestHandler.HandleDownloadRange`.

- Path parameter `range` is expected in the form `0-{N}`.
- Steps:
  1. Extract `rangePath := r.PathValue("range")`.
  2. Split by `-` and validate the `0-{N}` format.
  3. Parse `{N}` as `int64` (`endByte`) and ensure it is `< MaxPayloadBytes`.
  4. Compute `amount := endByte + 1` (bytes from 0 to N inclusive).
  5. Delegate to `ServeContent(w, r, amount)`, which:
     - Sets `Content-Type: application/octet-stream`.
     - Sets `Content-Length` to `amount`.
     - Opens `randomBufferFile` and uses `io.CopyN` from the file to the response writer, enabling the Go `net/http` server and kernel to use sendfile/zero-copy paths where supported.

This provides a large, deterministic stream suitable for the fast.com client’s bandwidth measurement logic while minimizing per-request CPU and allocations.

### Range-based Upload and Latency

Route: `POST /speedtest/range/{range}` → `HandleUploadRange`.

- Extract `rangePath := r.PathValue("range")`.
- If `rangePath != "0-0"` (upload test):
  - Drain the request body to `io.Discard`.
  - Close the body.
  - Respond with `200 OK`.
- If `rangePath == "0-0"` (latency test):
  - Do not read a body.
  - Immediately respond with `200 OK`.

This dual behavior matches the fast.com pattern, where `0-0` is used as a latency ping.

### Non-range Download and Upload

Routes:

- `GET /speedtest` → `SpeedTestHandler.HandleDownload`.
- `POST /speedtest` → `SpeedTestHandler.HandleUpload`.

Current status:

- **HandleDownload**:
  - Uses the same `ServeContent` helper as the range-based download.
  - Streams `MaxPayloadBytes` from `randomBufferFile` to the client with `Content-Length` set.

- **HandleUpload**:
  - Drains the request body to `io.Discard` and returns `200 OK`.

These endpoints complement the range-based endpoints and are implemented efficiently using the shared backing file.



## Telemetry Ingestion

### Purpose

Provide a local equivalent of Netflix’s `ichnaea` service (`https://ichnaea-web.netflix.com/cl2`) to accept logs from the fast.com JS logging module.

### Endpoint

- Route: `POST /telemetry/cl2` → `telemetry.HandleIngest`.
- The frontend JS bundle is patched to call this local path instead of the Netflix endpoint.

### Behavior

In `internal/telemetry/handler.go`:

- If `r.Body` is non-nil:
  - Defer `r.Body.Close()`.
  - Drain the body to `io.Discard` (we do not persist or process payloads in the MVP).
- Sets a number of response headers:
  - `Access-Control-Allow-Credentials: true`.
  - `Access-Control-Allow-Headers: ...` (long list of `X-Netflix-*` headers).
  - `Access-Control-Allow-Methods: GET, POST, OPTIONS`.
  - `Allow: GET, POST, OPTIONS`.
  - `X-Ichnaea`, `X-Content-Type-Options`, `X-XSS-Protection`, `Cache-Control`, `Pragma`, `Expires`, `X-Frame-Options`.
- Returns `200 OK` with empty body.

These headers mimic Netflix’s behavior closely enough to keep the logging library happy (no CORS or protocol errors).


## Static File Serving

The static web app is served from `cfg.StaticDir` (default `original-webapp`) via:

- `staticFS := http.FileServer(http.Dir(cfg.StaticDir))`.
- `router.Handle("/", staticFS)`.

Because this handler is registered last, it acts as a catch-all for any paths not matched by API routes, which is compatible with the SPA and its asset URLs.


## CORS and Security Considerations

- No CORS needed as we run from the same host.
- HTTPS is expected to be provided either by a reverse proxy (nginx, Caddy, Envoy) or by terminating TLS directly in front of the Go server; this is deployment-dependent and not handled in the current code.


## Extensibility and Future Work

The current design follows `docs/backend-working-plan.md` and sets a foundation for future phases:

- **OCA directory evolution**:
  - Load multiple OCAs from configuration (env or file).
  - Generate fully-qualified URLs using `BaseURL`.
  - Honor `urlCount`, `https`, and other query parameters.
  - Track OCA health and exclude unhealthy endpoints from responses.

- **Speed endpoint refinement**:
  - Implement `HandleDownload` and `HandleUpload` for `/speedtest`.
  - Add configurable payload sizes and patterns.
  - Optimize performance and resource usage under heavy load.

- **Localization and share endpoints**:
  - Add an `internal/localization` package to:
    - Serve `localized.json` explicitly if needed beyond static hosting.
    - Implement `/{lang}/share/{speed}{units}.html` share pages using templates.

- **Observability and operations**:
  - Add Prometheus metrics for request counts, latencies, and error rates.
  - Enhance structured logging with request IDs and client metadata.

The modular package layout (`config`, `server`, `oca`, `speed`, `telemetry`) is intended to make these extensions straightforward while keeping the core behavior close to the original fast.com expectations.
