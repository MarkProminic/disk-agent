package hostinfo

import (
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// scPhysPages is the _SC_PHYS_PAGES sysconf name on illumos.
const scPhysPages = 500

// osName reads the first line of /etc/release (e.g. "OmniOS v11 r151059").
func osName() string {
	raw, err := os.ReadFile("/etc/release")
	if err != nil {
		return "illumos"
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return "illumos"
}

// totalMemory multiplies the physical page count by the page size.
func totalMemory() uint64 {
	pages, err := unix.Sysconf(scPhysPages)
	if err != nil || pages < 0 {
		return 0
	}
	return uint64(pages) * uint64(unix.Getpagesize())
}
