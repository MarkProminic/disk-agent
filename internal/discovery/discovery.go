// Package discovery advertises the agent on the local network as the
// _disk-agent._tcp mDNS service the ha_disk_monitor integration discovers.
package discovery

import (
	"os"

	"github.com/grandcat/zeroconf"
)

// Service is the mDNS service type the integration matches.
const Service = "_disk-agent._tcp"

// Advertiser is one running advertisement.
type Advertiser struct {
	server *zeroconf.Server
}

// Advertise registers the agent under its hostname with the port it serves
// on, the scheme of that port and its version in the TXT record.
func Advertise(port int, scheme, version string) (*Advertiser, error) {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "disk-agent"
	}
	txt := []string{"hostname=" + hostname, "scheme=" + scheme, "version=" + version}
	server, err := zeroconf.Register(hostname, Service, "local.", port, txt, nil)
	if err != nil {
		return nil, err
	}
	return &Advertiser{server: server}, nil
}

// Shutdown withdraws the advertisement.
func (a *Advertiser) Shutdown() {
	if a != nil && a.server != nil {
		a.server.Shutdown()
	}
}
