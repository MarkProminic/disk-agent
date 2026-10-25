# Acknowledgments

Disk Agent is built using many excellent open-source projects and tools. We are grateful to the developers and communities behind these projects.

## Core Dependencies

### Runtime Dependencies

**goccy/go-yaml** - Pure-Go YAML parser and emitter

- Repository: [github.com/goccy/go-yaml](https://github.com/goccy/go-yaml)
- License: MIT
- Usage: Configuration file parsing

**golang.org/x/crypto** - Supplementary Go cryptography libraries

- Repository: [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto)
- License: BSD-3-Clause
- Usage: bcrypt hashing of API keys

**golang.org/x/sys** - Low-level operating system primitives

- Repository: [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys)
- License: BSD-3-Clause
- Usage: Host memory probe on illumos

**lumberjack** - Rolling log file writer

- Repository: [github.com/natefinch/lumberjack](https://github.com/natefinch/lumberjack)
- License: MIT
- Usage: Log file rotation

### Development and Release Tooling

**golangci-lint** - Go linter aggregator

- Website: [golangci-lint.run](https://golangci-lint.run/)
- License: GPL-3.0
- Usage: Static analysis and formatting enforcement

**govulncheck** - Go vulnerability scanner

- Repository: [golang.org/x/vuln](https://pkg.go.dev/golang.org/x/vuln)
- License: BSD-3-Clause
- Usage: Dependency vulnerability scanning in CI

**release-please** - Automated releases from conventional commits

- Repository: [github.com/googleapis/release-please](https://github.com/googleapis/release-please)
- License: Apache-2.0
- Usage: Versioning, changelog, and GitHub releases

## Platform and Ecosystem

**Go** - The programming language and standard library

- Website: [go.dev](https://go.dev/)
- License: BSD-3-Clause
- Usage: Core language and runtime (net/http, log/slog, embed, and more)

**smartmontools** - The SMART monitoring tools this agent reads

- Website: [smartmontools.org](https://www.smartmontools.org/)

**OpenZFS** and **illumos ZFS** - The pools this agent reports

- Websites: [openzfs.org](https://openzfs.org/), [illumos.org](https://illumos.org/)

**Home Assistant** - The home of the ha_disk_monitor integration this agent serves

- Website: [home-assistant.io](https://www.home-assistant.io/)

**GitHub** - Code hosting, issue tracking, CI/CD

- Website: [github.com](https://github.com/)

## Community and Inspiration

**Hyperweaver Agent** - The Go host-agent this repository's shape comes from

- Repository: [github.com/Makr91/hyperweaver-agent](https://github.com/Makr91/hyperweaver-agent)

**Zoneweaver Agent** - The OmniOS packaging and release pipeline this repository copies

- Repository: [github.com/Makr91/zoneweaver-agent](https://github.com/Makr91/zoneweaver-agent)

---

## Disclaimer

This acknowledgment file may not be exhaustive. If you believe a project or individual should be acknowledged here, please let us know by opening an issue or contributing to this file.

All trademarks and registered trademarks mentioned herein are the property of their respective owners.
