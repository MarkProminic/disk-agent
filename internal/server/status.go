package server

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/MarkProminic/disk-agent/internal/version"
)

type statusBrand struct {
	Name string `json:"name" example:"Disk Agent"`
	Repo string `json:"repo" example:"https://github.com/MarkProminic/disk-agent"`
}

var agentBrand = statusBrand{
	Name: "Disk Agent",
	Repo: "https://github.com/MarkProminic/disk-agent",
}

type statusPayload struct {
	Role     string      `json:"role" example:"disk-agent"`
	Brand    statusBrand `json:"brand"`
	Agent    string      `json:"agent"`
	Platform string      `json:"platform"`
	Arch     string      `json:"arch"`
	Version  string      `json:"version"`
	Hostname string      `json:"hostname"`
	Auth     []string    `json:"auth"`
	Features []string    `json:"features"`
	// Seconds since the agent started
	Uptime int64 `json:"uptime"`
}

var platformFeatures = []string{"health", "disks", "pools", "events"}

func archName() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return runtime.GOARCH
	}
}

// @Summary		Public identity and capabilities
// @Description	The public status probe. No authentication. auth is {apikey}; every other route takes the key as X-API-Key or Authorization: Bearer.
// @Tags			Status
// @Produce		json
// @Success		200	{object}	statusPayload	"Agent identity and capabilities"
// @Router			/api/status [get]
func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	writeJSON(w, statusPayload{
		Role:     "disk-agent",
		Brand:    agentBrand,
		Agent:    "disk-agent",
		Platform: runtime.GOOS,
		Arch:     archName(),
		Version:  version.Version,
		Hostname: hostname,
		Auth:     []string{"apikey"},
		Features: platformFeatures,
		Uptime:   int64(time.Since(s.startedAt).Seconds()),
	})
}
