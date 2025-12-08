# Fastclone

Self-hosted backend for the official https://fast.com speed test web app from Netflix.

This repo contains a backend for the fast.com web app and a patcher for the web app to use this backend. When building it as a container image, you can self host your own version of fast.com.


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

## Performance 

The most performance is achieved when the webbrowser can access the application without anything in the middle.

- A normal docker run will start a `docker-proxy` so data will travel from kernel (sendfile) -> user space for the proxy -> down again into the kernel to be sent to the network. You probably need to disable the `userland-proxy` in your `docker/daemon.json` for better performance.
- An ingress/loadbalancer/tls service that might sit in front of this app, will also cause performance issues.


### Interesting things to try/check.

- The right TCP kernel settings in linux for better performance.
- Good ingress alternatives to host this tool behind.
- Self hosted tls termination with ktls.

## Other notes

### AI usage

AI has been used to jumpstart and scaffold this project:
- Find insights and details in Netflix webapp code to build the backend.
- Scaffold the patch python script and the golang project.

### Copyright

The original webapp code that it is not committed here but will be used when building the container image is property of Netflix. That's why there are no complete builds of the final docker image, as I haven't asked for permission to redistribute the Netflix fast.com webapp.

The rest of the code in this repo uses the MIT License.