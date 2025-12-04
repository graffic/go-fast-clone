// Package telemetry handles the client telemetry ingestion endpoint.
package telemetry

import (
	"io"
	"net/http"
)

// HandleIngest handles POST /cl2 for telemetry data.
func HandleIngest(w http.ResponseWriter, r *http.Request) {
	// Drain the body to simulate reading it, as we are just a sink.
	// In a real implementation, we might process this JSON.
	if r.Body != nil {
		defer r.Body.Close()
		// Discard body
		_, _ = io.Copy(io.Discard, r.Body)
	}

	h := w.Header()

	h.Set("Access-Control-Allow-Credentials", "true")
	h.Set("Access-Control-Allow-Headers", "Accept,Accept-Language,Authorization,Content-Type,Content-Encoding,Cookie,debugRequest,X-Netflix.application.name,X-Netflix.application.version,X-Netflix.certification.version,X-Netflix.Client.Request.Name,X-Netflix.client.request.sendtime,X-Netflix.client.request.sendtimemono,X-Netflix.client.request.transport,X-Netflix.device.type,X-Netflix.esn,X-Netflix.ichnaea.request.type,X-Netflix.oauth.consumer.key,X-Netflix.oauth.token,X-Netflix.request.uuid,X-Netflix.user.id,X-Netflix.request.attempt,X-Netflix.request.id,X-Netflix.request.client.context,X-Netflix.request.client.sendtime,X-Netflix.request.client.sendtimemono")
	h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	h.Set("Allow", "GET, POST, OPTIONS")
	h.Set("X-Ichnaea", "~0=true~RL=382") // Custom header from original
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-XSS-Protection", "0")
	h.Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate")
	h.Set("Pragma", "no-cache")
	h.Set("Expires", "0")
	h.Set("X-Frame-Options", "DENY")

	// The original returns 200 OK with Content-Length: 0
	w.WriteHeader(http.StatusOK)
}
