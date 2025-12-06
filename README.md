# Fastclone

Self-hosted backend for the official https://fast.com speed test web app from Netflix


## Run Locally

**Prerequisites**
- Go 1.25+
- Python 3 (for updating the webapp assets)

**1. Fetch/patch the fast.com web app**

```bash
python3 scripts/update_webapp.py
```

This creates/updates the `original-webapp/` directory used for static files.

**2. Start the backend**

```bash
go run ./cmd/api
```

Then open: http://localhost:8080


## Run in a Container

The Docker image automatically downloads and patches the web app and embeds the static files.

**Build**

```bash
docker build -t fastclone .
```

**Run**

```bash
docker run --rm -p 8080:8080 fastclone
```

Then open: http://localhost:8080


## Configuration

Environment variables:
- `LISTEN_ADDR` – address/port to listen on (default `:8080`)
- `STATIC_DIR` – path to static files (default `original-webapp` in local runs; `/app/original-webapp` in the container)
- `HTTP_LOGGING` – enable HTTP request logging (default `false`)
- `ENABLE_PPROF` – enable debug/pprof endpoints (default `false`)
- `RANDOM_DATA_FILE` – path (preferably on tmpfs, default `/dev/shm/random_data`) used as the backing file for speed test payloads; populated on startup.

### Other notes

## AI usage

AI has been used to jumpstart and scaffold this project:
- Find insights and details in Netflix webapp code to build the backend.
- Scaffold the patch python script and the golang project.

## Copyright

The original webapp code that it is not committed here but will be used when building is property of Netflix.