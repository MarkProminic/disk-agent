# Disk Agent

**Disk Agent** serves the SMART state of every disk and the state of every ZFS pool of its host as JSON over HTTP and as a server-sent-event stream, for the [ha_disk_monitor](https://github.com/MarkProminic/ha_disk_monitor) Home Assistant integration. It is a single Go binary that runs under systemd on Debian and under SMF on OmniOS.

Nothing runs on a clock. The host's own daemons refresh the report through the agent's trigger socket: ZED on Linux and syseventd on illumos on every ZFS event, smartd on both on every SMART change it finds, udev and the device sysevents on hot plug. The agent advertises itself as `_disk-agent._tcp` over mDNS so Home Assistant offers it.

## Getting started

Debian, from the STARTcloud repository:

```bash
sudo apt install disk-agent
sudo systemctl enable --now disk-agent
sudo -u disk-agent disk-agent --config /etc/disk-agent key add homeassistant
```

OmniOS, from the STARTcloud repository:

```bash
pfexec pkg install system/monitoring/disk-agent
pfexec svcadm enable system/monitoring/disk-agent
pfexec su diskagent -c "/opt/disk-agent/disk-agent --config /etc/disk-agent key add homeassistant"
```

`key add` prints the API key once; paste it into the integration. `key list` and `key remove ID` manage them.

## Configuration

The configuration is a directory of two files: `app.config.yaml` and `auth.config.yaml`, read from `/etc/disk-agent` by both packages. `--config <dir>` or `CONFIG_DIR` names another.

Every key takes its default from the file's schema in `internal/config/schema/<name>.schema.yaml`.

Flags: `--config <dir>`, `--version`; subcommands `key add NAME`, `key list`, `key remove ID`.

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
