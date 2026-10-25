# Disk Agent

**Disk Agent** serves the SMART state of every disk and the state of every ZFS pool of its host as JSON over HTTP, for the [ha_disk_monitor](https://github.com/MarkProminic/ha_disk_monitor) Home Assistant integration to poll. It is a single Go binary that runs under systemd on Debian and under SMF on OmniOS.

## Getting started

Debian, from the STARTcloud repository:

```bash
sudo apt install disk-agent
sudo systemctl enable --now disk-agent
sudo cat /etc/disk-agent/setup.token
```

OmniOS, from the STARTcloud repository:

```bash
pfexec pkg install system/monitoring/disk-agent
pfexec svcadm enable system/monitoring/disk-agent
pfexec cat /etc/disk-agent/setup.token
```

The setup token is the proof of host ownership `POST /api/api-keys/bootstrap` demands for the first API key; the integration asks for it once and keeps the key it receives.

## Configuration

The configuration is a directory of two files: `app.config.yaml` and `auth.config.yaml`, read from `/etc/disk-agent` by both packages. `--config <dir>` or `CONFIG_DIR` names another.

Every key takes its default from the file's schema in `internal/config/schema/<name>.schema.yaml`; `GET /api/config/<name>/schema` serves the schema and `PUT /api/config/<name>` writes a merge patch.

Flags: `--config <dir>`, `--version`.

## Building from source

Requires Go 1.26 (the `go` line of `go.mod`).

```bash
go mod tidy
go build -o disk-agent .
```

Cross-compile for OmniOS from any OS:

```bash
GOOS=illumos GOARCH=amd64 CGO_ENABLED=0 go build -o disk-agent .
```

Packaging: [packaging/DEBIAN/README.md](packaging/DEBIAN/README.md) and [packaging/omnios/README.md](packaging/omnios/README.md).

## Contributing

Contributions welcome, see [CONTRIBUTING.md](CONTRIBUTING.md).

## License

GPL-3.0, see [LICENSE.md](LICENSE.md).
