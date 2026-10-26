package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/MarkProminic/disk-agent/internal/collector"
	"github.com/MarkProminic/disk-agent/internal/problem"
)

func writeEvent(w http.ResponseWriter, flusher http.Flusher, report *collector.Report) error {
	payload, err := json.Marshal(reportResponse{Hostname: hostname(), Report: report})
	if err != nil {
		return err
	}
	if _, werr := w.Write([]byte("event: report\ndata: ")); werr != nil {
		return werr
	}
	if _, werr := w.Write(payload); werr != nil {
		return werr
	}
	if _, werr := w.Write([]byte("\n\n")); werr != nil {
		return werr
	}
	flusher.Flush()
	return nil
}

// @Summary		The report stream
// @Description	Minimum role: viewer. A server-sent-event stream: the current report on connect as one report event, then a report event each time a trigger refreshes the state. The connection stays open until the client closes it.
// @Tags			Report
// @Produce		text/event-stream
// @Success		200	{string}	string	"event: report, data: the report"
// @Router			/api/events [get]
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		problem.Detail(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	reports, unsubscribe := s.collector.Subscribe()
	defer unsubscribe()

	if err := writeEvent(w, flusher, s.collector.Current()); err != nil {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case report := <-reports:
			if err := writeEvent(w, flusher, report); err != nil {
				slog.Debug("event stream closed", "remote", r.RemoteAddr, "error", err)
				return
			}
		}
	}
}
