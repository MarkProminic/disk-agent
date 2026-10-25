// Disk Agent serves the SMART state of every disk and the state of every ZFS
// pool of its host as JSON over HTTP for the ha_disk_monitor Home Assistant
// integration to poll.
//
//	@title			Disk Agent API
//	@version		1.0.0
//	@description	Disk Agent API v1. Capabilities are advertised by the public GET /api/status. Authorization is role-based (admin > operator > viewer) and enforced by a central method+path policy; each operation notes its minimum role.
//	@license.name	GPL-3.0
//	@license.url	https://github.com/MarkProminic/disk-agent/blob/main/LICENSE.md
//	@contact.name	Disk Agent
//	@contact.url	https://github.com/MarkProminic/disk-agent
//	@externalDocs.description	View on GitHub
//	@externalDocs.url	https://github.com/MarkProminic/disk-agent
//	@tag.name		Status
//	@tag.description	Public identity and capability discovery
//	@tag.name		API Keys
//	@tag.description	API-key lifecycle
//	@tag.name		System
//	@tag.description	Version and host information
//	@tag.name		Report
//	@tag.description	The SMART state of every disk and the state of every ZFS pool, collected when asked
//	@tag.name		Configuration
//	@tag.description	The two configuration files (app, auth) over the config contract's routes: the raw file, its JSON Schema, a merge-patch write refused with 422 and pointers, the pending restart list and the restart. Admin only.
//	@tag.name		Setup
//	@tag.description	First-run setup under the setup token in setup.token beside the configuration files.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MarkProminic/disk-agent/internal/auth"
	"github.com/MarkProminic/disk-agent/internal/config"
	"github.com/MarkProminic/disk-agent/internal/keys"
	"github.com/MarkProminic/disk-agent/internal/logging"
	"github.com/MarkProminic/disk-agent/internal/server"
	"github.com/MarkProminic/disk-agent/internal/version"
)

// main stays defer-free so os.Exit cannot skip cleanup; all work and all
// defers live in run.
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configDir := flag.String("config", "", "configuration directory (default: CONFIG_DIR, else the per-user config dir)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version.Version)
		return nil
	}

	cfg, err := config.Load(config.Options{Dir: *configDir, Seed: true})
	if err != nil {
		return err
	}
	resolvedPath := cfg.Dir()

	closeLog, err := logging.Setup(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := closeLog(); cerr != nil {
			fmt.Fprintln(os.Stderr, cerr)
		}
	}()

	slog.Info("disk-agent starting",
		"version", version.Version,
		"config", resolvedPath,
	)

	keyStore, err := keys.Open(cfg.KeyStorePath())
	if err != nil {
		slog.Error("key store setup failed", "error", err)
		return err
	}

	if cfg.APIKeys.BootstrapEnabled && cfg.APIKeys.BootstrapRequireClaimToken && keyStore.Count() == 0 {
		if token := auth.GetOrGenerateSetupToken(cfg.SetupTokenPath()); token != "" {
			slog.Info("Setup token (required to create the first API key): " + token)
		}
	}

	srv := server.New(cfg, keyStore)

	if lerr := srv.Listen(); lerr != nil {
		slog.Error("http server bind failed", "error", lerr)
		return lerr
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Start()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case srvErr := <-serverErr:
		if srvErr != nil {
			slog.Error("http server failed", "error", srvErr)
			return srvErr
		}
	case <-ctx.Done():
		slog.Info("signal received, shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if serr := srv.Shutdown(shutdownCtx); serr != nil {
		slog.Error("server shutdown", "error", serr)
	}
	slog.Info("disk-agent stopped")
	return nil
}
