// Disk Agent serves the SMART state of every disk and the state of every ZFS
// pool of its host as JSON over HTTP and as a server-sent-event stream for the
// ha_disk_monitor Home Assistant integration. The host's own ZED, sysevent,
// smartd and hot-plug hooks refresh the report through a unix socket; nothing
// runs on a clock.
//
//	@title			Disk Agent API
//	@version		1.0.0
//	@description	Disk Agent API v1. The public GET /api/status and GET /api/health need no credential; every other route takes an API key as X-API-Key or Authorization: Bearer. Keys are made on the host with disk-agent key add. GET /api/events streams a report on connect and on every change.
//	@license.name	GPL-3.0
//	@license.url	https://github.com/MarkProminic/disk-agent/blob/main/LICENSE.md
//	@contact.name	Disk Agent
//	@contact.url	https://github.com/MarkProminic/disk-agent
//	@externalDocs.description	View on GitHub
//	@externalDocs.url	https://github.com/MarkProminic/disk-agent
//	@tag.name		Status
//	@tag.description	Public identity and capability discovery
//	@tag.name		System
//	@tag.description	Version and host information
//	@tag.name		Report
//	@tag.description	The SMART state of every disk and the state of every ZFS pool, refreshed by the host's event hooks
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/MarkProminic/disk-agent/internal/config"
	"github.com/MarkProminic/disk-agent/internal/keys"
	"github.com/MarkProminic/disk-agent/internal/logging"
	"github.com/MarkProminic/disk-agent/internal/server"
	"github.com/MarkProminic/disk-agent/internal/version"
)

const usage = `usage:
  disk-agent [--config DIR]                  run the agent
  disk-agent [--config DIR] key add NAME     make an API key and print it once
  disk-agent [--config DIR] key list         list the API keys
  disk-agent [--config DIR] key remove ID    delete an API key
  disk-agent --version                       print the version`

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
	flag.Usage = func() { fmt.Fprintln(os.Stderr, usage) }
	flag.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version.Version)
		return nil
	}

	cfg, err := config.Load(config.Options{Dir: *configDir, Seed: true})
	if err != nil {
		return err
	}

	if flag.NArg() > 0 {
		return runKeyCommand(cfg, flag.Args())
	}
	return runAgent(cfg)
}

func runKeyCommand(cfg *config.Config, args []string) error {
	if args[0] != "key" || len(args) < 2 {
		return errors.New(usage)
	}
	store, err := keys.Open(cfg.KeyStorePath())
	if err != nil {
		return err
	}
	switch args[1] {
	case "add":
		if len(args) != 3 {
			return errors.New(usage)
		}
		apiKey, err := keys.GenerateKeyString(cfg.APIKeys.KeyLength)
		if err != nil {
			return err
		}
		if _, err := store.Create(apiKey, args[2], cfg.APIKeys.HashRounds); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, apiKey)
		return nil
	case "list":
		for _, k := range store.List() {
			fmt.Fprintf(os.Stdout, "%d\t%s\tcreated %s\tlast used %s\n", k.ID, k.Name, k.CreatedAt.Format(time.RFC3339), k.LastUsed.Format(time.RFC3339))
		}
		return nil
	case "remove":
		if len(args) != 3 {
			return errors.New(usage)
		}
		id, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return fmt.Errorf("key id %q is not a number", args[2])
		}
		k, err := store.Delete(id)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "removed %d %s\n", k.ID, k.Name)
		return nil
	}
	return errors.New(usage)
}

func runAgent(cfg *config.Config) error {
	closeLog, err := logging.Setup(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := closeLog(); cerr != nil {
			fmt.Fprintln(os.Stderr, cerr)
		}
	}()

	slog.Info("disk-agent starting", "version", version.Version, "config", cfg.Dir())

	keyStore, err := keys.Open(cfg.KeyStorePath())
	if err != nil {
		slog.Error("key store setup failed", "error", err)
		return err
	}
	if len(keyStore.List()) == 0 {
		slog.Warn("no API keys exist; make one with: disk-agent --config " + cfg.Dir() + " key add NAME")
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
