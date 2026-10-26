package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MarkProminic/disk-agent/internal/problem"
)

type triggerResponse struct {
	Kind        string `json:"kind"`
	Subclass    string `json:"subclass"`
	Device      string `json:"device"`
	CollectedAt string `json:"collected_at"`
	Ignored     bool   `json:"ignored"`
}

// ignoredSubclasses are the ZFS events that change nothing a report shows.
var ignoredSubclasses = map[string]bool{
	"history_event":      true,
	"config_cache_write": true,
}

func subclassOf(raw string) string {
	return strings.ToLower(strings.TrimPrefix(raw, "ESC_ZFS_"))
}

// handleTrigger refreshes the part of the report a host hook names. kind is
// zfs, smart or device; subclass is the ZFS event subclass; device is the
// device path a SMART event names. It is served on the trigger socket only.
func (s *Server) handleTrigger(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		problem.BadRequest(w)
		return
	}
	kind := r.FormValue("kind")
	subclass := subclassOf(r.FormValue("subclass"))
	device := r.FormValue("device")
	ctx := context.WithoutCancel(r.Context())
	response := triggerResponse{Kind: kind, Subclass: subclass, Device: device}
	switch kind {
	case "zfs":
		if ignoredSubclasses[subclass] {
			response.Ignored = true
			response.CollectedAt = s.collector.Current().CollectedAt
			writeJSON(w, response)
			return
		}
		response.CollectedAt = s.collector.CollectPools(ctx).CollectedAt
	case "smart":
		if device == "" {
			response.CollectedAt = s.collector.CollectDisks(ctx).CollectedAt
		} else {
			response.CollectedAt = s.collector.CollectDisk(ctx, device).CollectedAt
		}
	case "device":
		response.CollectedAt = s.collector.CollectDisks(ctx).CollectedAt
	case "all":
		response.CollectedAt = s.collector.CollectAll(ctx).CollectedAt
	default:
		problem.Invalid(w, problem.Enum("/kind", "zfs", "smart", "device", "all"))
		return
	}
	slog.Info("trigger", "kind", kind, "subclass", subclass, "device", device, "collected_at", response.CollectedAt)
	writeJSON(w, response)
}

// listenTrigger binds the trigger socket, a unix socket beside the
// configuration files that only the service user and root can reach.
func (s *Server) listenTrigger() error {
	path := s.cfg.TriggerSocketPath()
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(context.Background(), "unix", path)
	if err != nil {
		return err
	}
	if cerr := os.Chmod(path, 0o600); cerr != nil {
		_ = listener.Close()
		return cerr
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/trigger", s.handleTrigger)
	s.triggerListener = listener
	s.triggerSrv = &http.Server{Handler: recoverer(mux), ReadHeaderTimeout: 10 * time.Second}
	return nil
}
