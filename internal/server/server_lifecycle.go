package server

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/MarkProminic/disk-agent/internal/sslcert"
)

// Listen binds the configured address without serving yet. The HTTPS
// listener binds afterwards, non-fatally.
func (s *Server) Listen() error {
	listener, err := s.listen()
	if err != nil {
		return err
	}
	s.listener = listener
	s.listenHTTPS()
	return nil
}

// listenHTTPS binds the TLS listener when ssl.enabled: certificates are
// generated on demand (ssl.generate_ssl), and any certificate or bind problem
// logs an error and leaves the agent HTTP-only rather than failing startup.
func (s *Server) listenHTTPS() {
	if s.httpsSrv == nil {
		return
	}
	keyPath := s.cfg.SSLKeyPath()
	certPath := s.cfg.SSLCertPath()

	if s.cfg.SSL.GenerateSSL {
		generated, err := sslcert.EnsureCertificates(keyPath, certPath,
			s.cfg.SSLCACertPath(), s.cfg.SSLCAKeyPath())
		if err != nil {
			slog.Error("SSL certificate generation failed; HTTPS not started",
				"error", err, "key_path", keyPath, "cert_path", certPath)
			s.httpsSrv = nil
			return
		}
		if generated {
			slog.Info("SSL certificates generated (CA-signed)",
				"key_path", keyPath, "cert_path", certPath,
				"ca_cert_path", s.cfg.SSLCACertPath())
		}
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		slog.Error("SSL certificate error; HTTPS not started",
			"error", err, "key_path", keyPath, "cert_path", certPath)
		s.httpsSrv = nil
		return
	}

	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(context.Background(), "tcp", s.cfg.HTTPSListenAddr())
	if err != nil {
		slog.Error("https bind failed; HTTPS not started",
			"addr", s.cfg.HTTPSListenAddr(), "error", err)
		s.httpsSrv = nil
		return
	}
	s.httpsListener = tls.NewListener(listener, &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	})

	if s.cfg.SSL.ForceSecure {
		s.httpSrv.Handler = s.httpsRedirect()
	} else {
		slog.Info("ssl.force_secure is false: HTTP port keeps serving the full app alongside HTTPS",
			"http_addr", s.httpSrv.Addr, "https_addr", s.httpsSrv.Addr)
	}
}

// redirectHost returns the host the HTTPS redirect may target: the request's
// Host header is matched against the agent's own identities and the matched
// copy from the allowlist is returned, never the header value itself.
func (s *Server) redirectHost(requestHost string) string {
	host := requestHost
	if bare, _, err := net.SplitHostPort(host); err == nil {
		host = bare
	}
	allowed := []string{"127.0.0.1", "localhost", "::1", "[::1]", s.cfg.Server.BindAddress}
	if hostname, err := os.Hostname(); err == nil {
		allowed = append(allowed, hostname)
	}
	for _, candidate := range allowed {
		if candidate != "" && strings.EqualFold(host, candidate) {
			return candidate
		}
	}
	return "127.0.0.1"
}

// httpsRedirect sends every plain-HTTP request to its HTTPS counterpart.
func (s *Server) httpsRedirect() http.Handler {
	port := strconv.Itoa(s.cfg.Server.HTTPSPort)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := url.URL{
			Scheme:   "https",
			Host:     net.JoinHostPort(s.redirectHost(r.Host), port),
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
		http.Redirect(w, r, target.String(), http.StatusPermanentRedirect)
	})
}

// Start blocks serving HTTP until Shutdown is called or the listener fails.
// The HTTPS server (when up) serves on its own goroutine; its failure never
// takes the HTTP surface down.
func (s *Server) Start() error {
	if s.listener == nil {
		if err := s.Listen(); err != nil {
			return err
		}
	}

	if s.httpsSrv != nil && s.httpsListener != nil {
		slog.Info("https server listening", "addr", s.httpsSrv.Addr)
		go func() {
			serveErr := s.httpsSrv.Serve(s.httpsListener)
			if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				slog.Error("https server failed", "error", serveErr)
			}
		}()
	}

	slog.Info("http server listening", "addr", s.httpSrv.Addr)
	err := s.httpSrv.Serve(s.listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// listen binds the configured address once.
func (s *Server) listen() (net.Listener, error) {
	listenConfig := net.ListenConfig{}
	return listenConfig.Listen(context.Background(), "tcp", s.cfg.ListenAddr())
}

// Shutdown gracefully drains connections on both listeners.
func (s *Server) Shutdown(ctx context.Context) error {
	err := s.httpSrv.Shutdown(ctx)
	if s.httpsSrv != nil {
		if herr := s.httpsSrv.Shutdown(ctx); herr != nil {
			err = errors.Join(err, herr)
		}
	}
	return err
}
