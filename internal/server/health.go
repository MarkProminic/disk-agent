package server

import (
	"net/http"
	"time"

	"github.com/Makr91/disk-agent/internal/hostinfo"
)

const healthOK = "ok"

type healthReport struct {
	// ok | warning | error, the worst of the services
	Status string `json:"status" example:"ok"`
	// When the report was taken, RFC 3339
	Timestamp string `json:"timestamp"`
	// One coarse status word per service
	Services map[string]string `json:"services"`
	// The host as GET /api/version answers it
	Host hostinfo.Info `json:"host"`
}

func worstHealth(services map[string]string) string {
	worst := healthOK
	for _, state := range services {
		if state == "error" {
			return "error"
		}
		if state == "warning" {
			worst = "warning"
		}
	}
	return worst
}

func (s *Server) healthReport() *healthReport {
	services := map[string]string{
		"keys": healthOK,
	}
	return &healthReport{
		Status:    worstHealth(services),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  services,
		Host:      hostinfo.Get(),
	}
}

// @Summary		Health of the agent and its services
// @Description	Public. status is ok, warning or error, the worst of the services; timestamp is when the report was taken; services carries one coarse status word each; host is the host as GET /api/version answers it.
// @Tags			Status
// @Produce		json
// @Success		200	{object}	healthReport	"The health report"
// @Router			/api/health [get]
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.healthReport())
}
