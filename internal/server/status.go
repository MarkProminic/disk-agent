package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/Makr91/disk-agent/internal/config"
	"github.com/Makr91/disk-agent/internal/version"
)

type statusBrand struct {
	Name string `json:"name" example:"Disk Agent"`
	Repo string `json:"repo" example:"https://github.com/Makr91/disk-agent"`
}

var agentBrand = statusBrand{
	Name: "Disk Agent",
	Repo: "https://github.com/Makr91/disk-agent",
}

type statusPayload struct {
	Role               string      `json:"role" example:"disk-agent"`
	Brand              statusBrand `json:"brand"`
	Config             []string    `json:"config"`
	Agent              string      `json:"agent"`
	Platform           string      `json:"platform"`
	Arch               string      `json:"arch"`
	Version            string      `json:"version"`
	Hostname           string      `json:"hostname"`
	Auth               []string    `json:"auth"`
	BootstrapAvailable bool        `json:"bootstrapAvailable"`
	Features           []string    `json:"features"`
	// Seconds since the agent started
	Uptime int64 `json:"uptime"`
}

// platformFeatures are the capability tokens this agent always advertises.
var platformFeatures = []string{"health", "admin", "setup", "disks", "pools"}

// archName maps Go architecture names to the contract values.
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
// @Description	The public status probe. No authentication. role is the package name, disk-agent. config lists the configuration files (app, auth) the /api/config routes serve; features lists the capability tokens; auth is {apikey}.
// @Tags			Status
// @Produce		json
// @Success		200	{object}	statusPayload	"Agent identity and capabilities"
// @Router			/api/status [get]
func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	akCfg := s.cfg.APIKeys
	bootstrapAvailable := akCfg.BootstrapEnabled &&
		(s.keys.Count() == 0 || !akCfg.BootstrapAutoDisable)

	payload := statusPayload{
		Role:               "disk-agent",
		Brand:              agentBrand,
		Config:             config.Names,
		Agent:              "disk-agent",
		Platform:           runtime.GOOS,
		Arch:               archName(),
		Version:            version.Version,
		Hostname:           hostname,
		Auth:               []string{"apikey"},
		BootstrapAvailable: bootstrapAvailable,
		Features:           platformFeatures,
		Uptime:             int64(time.Since(s.startedAt).Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("write status response", "error", err)
	}
}
