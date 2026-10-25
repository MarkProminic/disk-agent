// Package config loads and provides the agent's YAML configuration.
package config

// LoggingConfig controls slog output.
type LoggingConfig struct {
	Level      string `yaml:"level"       json:"level"`
	Console    bool   `yaml:"console"     json:"console"`
	File       string `yaml:"file"        json:"file"`
	MaxSizeMB  int    `yaml:"max_size_mb" json:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	// Compression gzips rotated log files.
	Compression bool `yaml:"compression" json:"compression"`
	// Categories overrides the level per log category. Categories this agent
	// emits: app (the default), api_requests, auth.
	Categories map[string]string `yaml:"categories" json:"categories"`
}

// CollectionConfig names the tools the collector runs and the privilege
// command that wraps them.
type CollectionConfig struct {
	SmartctlPath string `yaml:"smartctl_path" json:"smartctl_path"`
	ZpoolPath    string `yaml:"zpool_path"    json:"zpool_path"`
	// PrivilegeCommand is prepended to every tool call: sudo on Debian,
	// pfexec on OmniOS, empty when the agent runs as root.
	PrivilegeCommand string `yaml:"privilege_command" json:"privilege_command"`
}
