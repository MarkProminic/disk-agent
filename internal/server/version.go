package server

import (
	"net/http"
	"runtime"
	"time"

	"github.com/MarkProminic/disk-agent/internal/hostinfo"
	"github.com/MarkProminic/disk-agent/internal/version"
)

type versionResponse struct {
	Success       bool          `json:"success"`
	Message       string        `json:"message"`
	Timestamp     string        `json:"timestamp"`
	Version       string        `json:"version"`
	Name          string        `json:"name"`
	GoVersion     string        `json:"go_version"`
	Platform      string        `json:"platform"`
	Arch          string        `json:"arch"`
	UptimeSeconds int64         `json:"uptime_seconds"`
	Host          hostinfo.Info `json:"host"`
}

// @Summary		Application version and host environment
// @Description	Minimum role: viewer.
// @Tags			System
// @Produce		json
// @Success		200	{object}	versionResponse	"Version information"
// @Router			/api/version [get]
func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, versionResponse{
		Success:       true,
		Message:       "Version information retrieved",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Version:       version.Version,
		Name:          "disk-agent",
		GoVersion:     runtime.Version(),
		Platform:      runtime.GOOS,
		Arch:          runtime.GOARCH,
		UptimeSeconds: int64(time.Since(s.startedAt).Seconds()),
		Host:          hostinfo.Get(),
	})
}
