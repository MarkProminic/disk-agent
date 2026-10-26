package server

import (
	"net/http"
	"os"

	"github.com/MarkProminic/disk-agent/internal/collector"
	"github.com/MarkProminic/disk-agent/internal/disks"
	"github.com/MarkProminic/disk-agent/internal/pools"
)

type reportResponse struct {
	Hostname string `json:"hostname"`
	*collector.Report
}

type disksResponse struct {
	Hostname    string             `json:"hostname"`
	CollectedAt string             `json:"collected_at"`
	Disks       []disks.Disk       `json:"disks"`
	Unreadable  []disks.Unreadable `json:"unreadable"`
	Error       *string            `json:"error"`
}

type poolsResponse struct {
	Hostname    string       `json:"hostname"`
	CollectedAt string       `json:"collected_at"`
	Pools       []pools.Pool `json:"pools"`
	Error       *string      `json:"error"`
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

// @Summary		Every disk and pool of the host
// @Description	Minimum role: viewer. The current report: every disk keyed by serial and every pool by name, with the time of the last refresh. The report is refreshed by the host's own event hooks, never by this request. A scanned device that answers no serial, a stale node of a pulled disk among them, is listed under unreadable with its error. disk_error and pool_error carry the failure of a whole collection.
// @Tags			Report
// @Produce		json
// @Success		200	{object}	reportResponse	"The report"
// @Router			/api/report [get]
func (s *Server) handleReport(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, reportResponse{Hostname: hostname(), Report: s.collector.Current()})
}

// @Summary		Every disk of the host
// @Description	Minimum role: viewer. The disks member of GET /api/report.
// @Tags			Report
// @Produce		json
// @Success		200	{object}	disksResponse	"The disks"
// @Router			/api/disks [get]
func (s *Server) handleDisks(w http.ResponseWriter, _ *http.Request) {
	report := s.collector.Current()
	writeJSON(w, disksResponse{Hostname: hostname(), CollectedAt: report.CollectedAt, Disks: report.Disks, Unreadable: report.Unreadable, Error: report.DiskError})
}

// @Summary		Every pool of the host
// @Description	Minimum role: viewer. The pools member of GET /api/report.
// @Tags			Report
// @Produce		json
// @Success		200	{object}	poolsResponse	"The pools"
// @Router			/api/pools [get]
func (s *Server) handlePools(w http.ResponseWriter, _ *http.Request) {
	report := s.collector.Current()
	writeJSON(w, poolsResponse{Hostname: hostname(), CollectedAt: report.CollectedAt, Pools: report.Pools, Error: report.PoolError})
}
