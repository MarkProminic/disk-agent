// Package config loads and provides the agent's YAML configuration.
package config

// ServerConfig controls the HTTP and HTTPS listeners.
type ServerConfig struct {
	BindAddress string `yaml:"bind_address" json:"bind_address"`
	Port        int    `yaml:"port"         json:"port"`
	// HTTPSPort is the TLS listener's port; bound only when ssl.enabled.
	HTTPSPort int `yaml:"https_port" json:"https_port"`
}

// SSLConfig controls the agent's HTTPS listener: certificate problems never
// stop the agent, HTTPS is skipped with an error in the log and HTTP keeps
// serving.
type SSLConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	// ForceSecure makes the plain-HTTP port serve only 308 redirects once the
	// TLS listener is up; false keeps the HTTP port serving the full app
	// alongside HTTPS.
	ForceSecure bool `yaml:"force_secure" json:"force_secure"`
	// GenerateSSL creates the server certificate at the paths below when none
	// exists, signed by the CA at the CA paths, itself generated when absent.
	GenerateSSL bool `yaml:"generate_ssl" json:"generate_ssl"`
	// KeyPath/CertPath locate the server private key and certificate. Empty
	// selects <config dir>/ssl/server.key and <config dir>/ssl/server.crt.
	KeyPath  string `yaml:"key_path"  json:"key_path"`
	CertPath string `yaml:"cert_path" json:"cert_path"`
	// CACertPath/CAKeyPath locate the CA used to sign the generated server
	// certificate. Empty selects <config dir>/ssl/ca.crt and
	// <config dir>/ssl/ca.key.
	CACertPath string `yaml:"ca_cert_path" json:"ca_cert_path"`
	CAKeyPath  string `yaml:"ca_key_path"  json:"ca_key_path"`
}

// CORSConfig controls Cross-Origin Resource Sharing: the API key, not the
// browser Origin, is the access boundary, so allow_all defaults to true and
// allow_all: false falls back to the explicit whitelist.
type CORSConfig struct {
	AllowAll  bool     `yaml:"allow_all" json:"allow_all"`
	Whitelist []string `yaml:"whitelist" json:"whitelist"`
}

// APIKeysConfig controls how API keys are hashed and sized.
type APIKeysConfig struct {
	HashRounds int `yaml:"hash_rounds" json:"hash_rounds"`
	KeyLength  int `yaml:"key_length"  json:"key_length"`
}
