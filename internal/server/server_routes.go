package server

import (
	"net/http"

	"github.com/Makr91/disk-agent/internal/auth"
	"github.com/Makr91/disk-agent/internal/configengine"
	"github.com/Makr91/disk-agent/internal/problem"
)

func handleUnknownAPI(w http.ResponseWriter, _ *http.Request) {
	problem.NotFound(w)
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/", handleUnknownAPI)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/health", s.handleHealth)

	requireKey := auth.Middleware(s.keys)
	mux.HandleFunc("POST /api/api-keys/bootstrap", s.handleBootstrapKey)
	mux.Handle("POST /api/api-keys/generate", requireKey(http.HandlerFunc(s.handleGenerateKey)))
	mux.Handle("GET /api/api-keys", requireKey(http.HandlerFunc(s.handleListKeys)))
	mux.Handle("GET /api/api-keys/info", requireKey(http.HandlerFunc(s.handleKeyInfo)))
	mux.Handle("DELETE /api/api-keys/{id}", requireKey(http.HandlerFunc(s.handleDeleteKey)))
	mux.Handle("PUT /api/api-keys/{id}/revoke", requireKey(http.HandlerFunc(s.handleRevokeKey)))

	mux.Handle("GET /api/version", requireKey(http.HandlerFunc(s.handleVersion)))
	mux.Handle("GET /api/report", requireKey(http.HandlerFunc(s.handleReport)))
	mux.Handle("GET /api/disks", requireKey(http.HandlerFunc(s.handleDisks)))
	mux.Handle("GET /api/pools", requireKey(http.HandlerFunc(s.handlePools)))

	s.cfg.Engine().Routes(mux, configengine.Auth{
		Admin: requireKey,
		Actor: func(r *http.Request) string {
			if identity := auth.FromContext(r.Context()); identity != nil {
				return identity.Name
			}
			return "setup"
		},
		User: "disk-agent",
	}, s.exit, 1<<20)
}
