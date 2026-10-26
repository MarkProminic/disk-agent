package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/MarkProminic/disk-agent/internal/auth"
	"github.com/MarkProminic/disk-agent/internal/problem"
)

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("write json response", "error", err)
	}
}

func handleUnknownAPI(w http.ResponseWriter, _ *http.Request) {
	problem.NotFound(w)
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/", handleUnknownAPI)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/health", s.handleHealth)

	requireKey := auth.Middleware(s.keys)
	mux.Handle("GET /api/version", requireKey(http.HandlerFunc(s.handleVersion)))
	mux.Handle("GET /api/report", requireKey(http.HandlerFunc(s.handleReport)))
	mux.Handle("GET /api/disks", requireKey(http.HandlerFunc(s.handleDisks)))
	mux.Handle("GET /api/pools", requireKey(http.HandlerFunc(s.handlePools)))
}
