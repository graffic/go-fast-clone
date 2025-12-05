# Goal

- To build a selfhosted clone for a speed test web app called fast.com (https://fast.com)
- It will use the official fast.com web app, that will be downloaded and patched to work with our backend.
- Create a new backend for the official web app, so the app can be served locally.
- Reuse as many external assets as possible. (this might change in the future and we will host everything)
- The entire system: backend and frontend will be packed in a container so it can be easily deployed.

## Frontend

- It is created from the available fast.com webpage via a python script `scripts/update_webapp.py`.
- Keep the minimal amount of files local, starting from `index.html`.
- The files we need to keep local are:
  - the main app JS and CSS bundle (for example `/app-*.(js|css)`),
  - the localization JSON file(s) referenced from the page (for example `/localized*.json` for the supported languages).
- All other assets (fonts, favicons, manifest, share images, etc.) should continue to be loaded from the original `fast.com` URLs unless or until we decide to self-host everything.
- Patch links and references so locally served pages still use the correct resources (local bundles and/or the original `fast.com` assets as intended).

## Backend

- The main binary lives in `cmd/api` and uses packages under `internal/`.
- Configuration is read from env vars via `internal/config` (`LISTEN_ADDR`, `STATIC_DIR`).
- HTTP routing and middleware (logging if enabled) are defined in `internal/server` and should remain minimal and composable.
- Speed test logic lives in `internal/speed` (download/upload/latency endpoints); prefer efficient streaming (`io.CopyN`, shared buffers) over per-request allocation.
- The OCA directory handler in `internal/oca` should expose `/netflix/speedtest/v2` with a response shape compatible with the upstream fast.com app.
- Telemetry ingestion lives in `internal/telemetry` and should mimic Netflix `ichnaea` behaviour enough for the JS logger to succeed (status code + headers), but can discard bodies by default.
