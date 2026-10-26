// Package server hosts the agent's HTTP surface.
package server

import (
	"net"
	"net/http"
	"time"

	"github.com/MarkProminic/disk-agent/internal/collector"
	"github.com/MarkProminic/disk-agent/internal/config"
	"github.com/MarkProminic/disk-agent/internal/discovery"
	"github.com/MarkProminic/disk-agent/internal/disks"
	"github.com/MarkProminic/disk-agent/internal/keys"
	"github.com/MarkProminic/disk-agent/internal/pools"
)

// Server is the agent's HTTP (and optional HTTPS) server, its trigger socket
// and its mDNS advertisement.
type Server struct {
	cfg       *config.Config
	keys      *keys.Store
	collector *collector.Collector
	httpSrv   *http.Server
	listener  net.Listener
	startedAt time.Time

	// httpsSrv/httpsListener exist only when ssl.enabled AND the certificate
	// loaded; certificate problems leave the agent HTTP-only, never down.
	httpsSrv      *http.Server
	httpsListener net.Listener

	triggerSrv      *http.Server
	triggerListener net.Listener

	advertiser *discovery.Advertiser
}

// New builds the server and its routes.
func New(cfg *config.Config, keyStore *keys.Store) *Server {
	s := &Server{
		cfg:  cfg,
		keys: keyStore,
		collector: &collector.Collector{
			Disks: disks.Runner{Smartctl: cfg.Collection.SmartctlPath, Privilege: cfg.Collection.PrivilegeCommand},
			Pools: pools.Runner{Zpool: cfg.Collection.ZpoolPath, Privilege: cfg.Collection.PrivilegeCommand},
		},
		startedAt: time.Now(),
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)
	handler := requestLog(recoverer(corsMiddleware(&cfg.CORS, mux)))
	s.httpSrv = &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if cfg.SSL.Enabled {
		s.httpsSrv = &http.Server{
			Addr:              cfg.HTTPSListenAddr(),
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		}
	}
	return s
}
