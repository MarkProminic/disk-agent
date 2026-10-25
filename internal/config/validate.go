package config

import (
	"net"

	"github.com/Makr91/disk-agent/internal/problem"
)

func failure(pointer, rule string, params map[string]any) []problem.Error {
	if params == nil {
		params = map[string]any{}
	}
	return []problem.Error{{Pointer: pointer, Rule: rule, Params: params}}
}

func stringValue(value any) (string, bool) {
	s, ok := value.(string)
	return s, ok
}

func (c *Config) codeRules(pointer string, value any, name string, _ map[string]any, _ string) []problem.Error {
	if name == "app" && pointer == "/server/bind_address" {
		if s, ok := stringValue(value); ok && s != "" && net.ParseIP(s) == nil {
			return failure(pointer, "format", map[string]any{"format": "ip"})
		}
	}
	return nil
}
