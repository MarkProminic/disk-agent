// Package discovery advertises the agent on the local network as the
// _disk-agent._tcp mDNS service the ha_disk_monitor integration discovers.
package discovery

import (
	"net"
	"os"

	"github.com/grandcat/zeroconf"
)

// Service is the mDNS service type the integration matches.
const Service = "_disk-agent._tcp"

// Advertiser is one running advertisement.
type Advertiser struct {
	server *zeroconf.Server
}

// interfaces answers every interface that is up, not loopback and holds an
// IPv4 address. The list is passed explicitly because illumos reports no
// multicast flag on its interfaces and the library would otherwise join the
// mDNS group on none of them, leaving the agent unable to answer a browse.
func interfaces() []net.Interface {
	all, err := net.Interfaces()
	if err != nil {
		return nil
	}
	out := []net.Interface{}
	for _, iface := range all {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, aerr := iface.Addrs()
		if aerr != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if ok && ipnet.IP.To4() != nil {
				out = append(out, iface)
				break
			}
		}
	}
	return out
}

// Advertise registers the agent under its hostname with the port it serves
// on, the scheme of that port and its version in the TXT record, on every
// interface that carries an IPv4 address.
func Advertise(port int, scheme, version string) (*Advertiser, error) {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "disk-agent"
	}
	txt := []string{"hostname=" + hostname, "scheme=" + scheme, "version=" + version}
	server, err := zeroconf.Register(hostname, Service, "local.", port, txt, interfaces())
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
