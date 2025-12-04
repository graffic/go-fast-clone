# Fast.com Clone – Frontend Design

## Overview

The frontend is a lightly patched clone of the official `fast.com` single-page app. It is periodically synchronized from `https://fast.com/` by `scripts/update_webapp.py` into `original-webapp/`, and then served by the Go backend at `/`.

Design goals:
- Reuse as much of the upstream fast.com app as possible.
- Keep local copies only of assets that must be patched or cannot be used directly due to CORS.
- Patch the web app so that it talks to the local backend instead of Netflix infrastructure.


## Asset Fetching and Layout

All synchronization and patching is driven by `scripts/update_webapp.py`.

- **Source origin**: `FAST_ROOT_URL = "https://fast.com/"`.
- **Destination directory**: `WEBAPP_DIR = CWD / "original-webapp"`.
- On each run:
  - `original-webapp/` is removed (`shutil.rmtree`) and recreated.
  - `index.html` is fetched from `https://fast.com/` to `original-webapp/index.html`.
  - Core app bundles and localization JSON are discovered by parsing `index.html`.
  - A fixed set of CORS-sensitive font assets are downloaded.

### HTML Parsing for Core Assets

`AppAssetRefs` is an `HTMLParser` subclass that locates the key local asset references in `index.html`:

- Tracks three paths:
  - `js_path` – app JS bundle (`/app-*.js`).
  - `css_path` – app CSS bundle (`/app-*.css`).
  - `localized_path` – localization JSON (`/localized*.json`).
- Parsing rules:
  - `<script src="/app-*.js">` → JS bundle.
  - `<link href="/app-*.css">` → CSS bundle.
  - `<body localized="/localized-*.json">` → localization JSON.
- `validate()` enforces that all three are found, otherwise the script fails early.

The located asset URLs are then fetched from `https://fast.com/` and written into `original-webapp/` using `http_get` and `fetch_asset`.

### Fixed CORS-Sensitive Assets

Because the `oc-webfont` family does not have CORS headers suitable for direct cross-origin loading, a minimal set of font-related assets is always downloaded locally:

- `/assets/fonts/oc-webfont.min.css`
- `/assets/fonts/fonts/oc-webfont.svg`
- `/assets/fonts/fonts/oc-webfont.ttf`
- `/assets/fonts/fonts/oc-webfont.woff`
- `/assets/fonts/fonts/oc-webfont.eot`

These are stored under `original-webapp/assets/fonts/...` and later referenced via patched URLs.


## HTTP Fetch Logic

All upstream asset fetches are done through small helpers:

- `http_get(url, destination)`
  - Issues an HTTP GET with a custom `User-Agent` (`fast-clone-fetcher/0.1`).
  - Validates `resp.status == 200`.
  - Writes the raw bytes to `destination`.
  - Returns the response decoded as UTF‑8 (with replacement on errors).

- `fetch_asset(asset_path, dest_dir)`
  - Computes the absolute URL via `urljoin(FAST_ROOT_URL, asset_path)`.
  - Writes the file into `dest_dir / basename(asset_path)`.

These helpers centralize error handling and logging for asset synchronization.


## Patching Strategy

After assets have been fetched, `update_webapp.py` applies targeted patches to the main JS bundle, the CSS bundle, and `index.html`. The goal is to:

- Redirect API calls from Netflix endpoints to local backend routes.
- Keep most visual/branding assets hosted on `fast.com`.
- Self-host only fonts and core app bundles.

### JS Bundle Patching – API Endpoints

The JS bundle (e.g., `app-<hash>.js`) is patched by `patch_file(js_file, [...])` to rewrite backend endpoints and URL construction:

- Replace Netflix API hostnames with local paths:
  - `"api.fast.com/netflix/speedtest/v2"` → `"/netflix/speedtest/v2"`
  - `"api-global.netflix.com/oca/speedtest"` → `"/api/oca/speedtest"`
- Redirect telemetry endpoints to local equivalents:
  - `"https://ichnaea-web.netflix.com/cl2"` → `"/telemetry/cl2"`
  - `"https://ichnaea.test.netflix.com/cl2"` → `"/telemetry-test/cl2"`
- Make URL construction backend-agnostic:
  - `url="https://"+endpoint+"` → `url=endpoint+"`

Effectively, all network traffic from the web app is directed to the local Go services while preserving the original app logic and request shapes.

### CSS Bundle Patching – Asset URLs

The CSS bundle is patched to keep most non-font assets served directly from `fast.com`:

- `"url(/assets/"` → `"url(https://fast.com/assets/"`

This means background images, icons, and other visuals referenced in CSS continue to be loaded from the upstream site, minimizing local storage and update complexity.

### index.html Patching – Mixed Local/Remote Assets

`index.html` is patched to:

- Keep most assets remote, but fonts local:
  - `"=/assets/"` → `"=https://fast.com/assets/"` (default asset references).
  - `"=\"https://fast.com/assets/fonts/"` → `"=/assets/fonts/"` (font CSS and files now served locally from `original-webapp`).
- Adjust canonical URL for self-hosted deployments:
  - `href="https://fast.com" rel="canonical"` → `href="/" rel="canonical"`.

The script intentionally does **not** change the app JS/CSS bundle references, which are already relative paths (`/app-*.js`, `/app-*.css`) that work locally.


## Runtime Integration with Backend

At runtime, the Go backend (see `internal/server/server.go`) serves `original-webapp/` at the root path `/` via `http.FileServer`. The patched JS bundle then uses local endpoints for API calls:

- `GET /netflix/speedtest/v2` – OCA directory endpoint (implemented in `internal/oca`).
- `GET /speedtest` and `GET /speedtest/range/0-{N}` – download endpoints (implemented in `internal/speed`).
- `POST /speedtest` and `POST /speedtest/range/0-{N}` – upload and latency endpoints.
- `POST /telemetry/cl2` – telemetry ingestion endpoint (implemented in `internal/telemetry`).

Localization JSON (`/localized*.json`) is loaded from the local `original-webapp/` directory, as indicated by the `localized` attribute in the `<body>` element of `index.html`.


## Operations

- To refresh the frontend to the latest fast.com version:
  - From the repository root, run: `./scripts/update_webapp.py` (or `python3 scripts/update_webapp.py`).
- The script is safe to re-run; it fully rebuilds `original-webapp/` each time.


## Extensibility

The frontend design intentionally keeps divergence from upstream minimal. Future enhancements can be layered on top:

- **More locales**: capture and preserve additional `localized-*.json` files if fast.com adds language variants.
- **Self-host all assets**: extend `FIXED_ASSETS` and adjust CSS/HTML rewrites to download and serve more assets locally when desired.
- **Configurable upstream**: make `FAST_ROOT_URL` configurable (e.g., to point to a mirror or archived snapshot).
- **Custom branding**: add optional patches to index/ CSS for branding, while retaining the same functional behavior.
