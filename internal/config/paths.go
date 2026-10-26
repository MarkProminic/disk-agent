package config

import (
	"path/filepath"
)

// SSLKeyPath returns the TLS private key location: ssl.key_path when
// configured, else ssl/server.key beside the loaded configuration file.
func (c *Config) SSLKeyPath() string {
	if c.SSL.KeyPath != "" {
		return c.SSL.KeyPath
	}
	return filepath.Join(c.dir, "ssl", "server.key")
}

// SSLCertPath returns the TLS certificate location: ssl.cert_path when
// configured, else ssl/server.crt beside the loaded configuration file.
func (c *Config) SSLCertPath() string {
	if c.SSL.CertPath != "" {
		return c.SSL.CertPath
	}
	return filepath.Join(c.dir, "ssl", "server.crt")
}

// SSLCACertPath returns the CA certificate location: ssl.ca_cert_path when
// configured, else ssl/ca.crt beside the loaded configuration file.
func (c *Config) SSLCACertPath() string {
	if c.SSL.CACertPath != "" {
		return c.SSL.CACertPath
	}
	return filepath.Join(c.dir, "ssl", "ca.crt")
}

// SSLCAKeyPath returns the CA private-key location: ssl.ca_key_path when
// configured, else ssl/ca.key beside the loaded configuration file.
func (c *Config) SSLCAKeyPath() string {
	if c.SSL.CAKeyPath != "" {
		return c.SSL.CAKeyPath
	}
	return filepath.Join(c.dir, "ssl", "ca.key")
}

// LogFilePath returns the configured log file, defaulting to
// <config dir>/logs/agent.log.
func (c *Config) LogFilePath() (string, error) {
	if c.Logging.File != "" {
		return c.Logging.File, nil
	}
	return filepath.Join(c.dir, "logs", "agent.log"), nil
}

// KeyStorePath returns the API-key store location: keys.json beside the
// configuration files.
func (c *Config) KeyStorePath() string {
	return filepath.Join(c.dir, "keys.json")
}
