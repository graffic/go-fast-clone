// Package oca handles the OCA (Open Connect Appliance) directory endpoint.
// This endpoint returns a list of speed test server URLs to the webapp.
package oca

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// Location represents a geographic location.
type Location struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

// Target represents a speed test server target.
type Target struct {
	Name     string   `json:"name"`
	URL      string   `json:"url"`
	Location Location `json:"location"`
	ID       string   `json:"id,omitempty"`
	Label    string   `json:"label,omitempty"`
}

// ClientInfo represents information about the client.
type ClientInfo struct {
	IP       string   `json:"ip"`
	ASN      string   `json:"asn"`
	ISP      string   `json:"isp,omitempty"`
	Location Location `json:"location"`
}

// DirectoryResponse is the response shape for GET /netflix/speedtest/v2.
type DirectoryResponse struct {
	Targets []Target   `json:"targets"`
	Client  ClientInfo `json:"client"`
}

// HandleDirectory handles GET /netflix/speedtest/v2.
func HandleDirectory(w http.ResponseWriter, r *http.Request) {
	baseURL := "/speedtest?e=" + strconv.FormatInt(time.Now().UnixMilli(), 10)
	location := Location{
		City:    "LocalCity",
		Country: "LC",
	}

	response := DirectoryResponse{
		Client: ClientInfo{
			IP:       r.RemoteAddr,
			ASN:      "65535", // Dummy ASN
			Location: location,
		},
		Targets: []Target{
			{
				Name:     baseURL,
				URL:      baseURL,
				Location: location,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
