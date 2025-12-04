# Step 0: Patch the webapp
FROM python:3.13-alpine AS webapp-patcher

WORKDIR /app

COPY scripts/update_webapp.py .

RUN python update_webapp.py

# Step 1: Build the application
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the binary
# CGO_ENABLED=0 is important for distroless static images
RUN CGO_ENABLED=0 GOOS=linux go build -o /fastclone cmd/api/main.go

# Step 2: Create the final image
FROM gcr.io/distroless/static-debian12

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /fastclone /app/fastclone

# Copy the static files
COPY --from=webapp-patcher /app/original-webapp /app/original-webapp

# Set environment variable for static directory
ENV STATIC_DIR=/app/original-webapp

# Expose the port
EXPOSE 8080

ENTRYPOINT ["/app/fastclone"]

