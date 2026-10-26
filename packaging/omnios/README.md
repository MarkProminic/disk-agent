# Disk Agent - OmniOS Package

This directory contains the files needed to build an OmniOS IPS package for the Disk Agent.

## Package Information

- **Package Name**: `system/monitoring/disk-agent`
- **Service Name**: `system/monitoring/disk-agent`
- **User/Group**: `diskagent` (UID/GID: 302)
- **Installation Path**: `/opt/disk-agent`
- **Configuration**: `/etc/disk-agent`
- **Data Directory**: `/var/lib/disk-agent` (also user home directory)
- **Log Directory**: `/var/log/disk-agent`

## Package Contents

### Build Files

- `build.sh` - Main build script that builds the Go binary and creates the IPS package
- `disk-agent.p5m` - IPS package manifest

### SMF Service Files

- `disk-agent-smf.xml` - SMF service manifest (stop method uses SMF's `:kill` token)
- `startup.sh` - Service startup script
- `disk-agent.user_attr` - RBAC profile grant for the service user

### Configuration

- `app.config.yaml` - OmniOS seed naming `/opt/ooce/sbin/smartctl`, `/usr/sbin/zpool` and `pfexec`
- `../../internal/config/seed/auth.config.yaml` - Seed authentication file

## Dependencies

- **smartmontools**: `ooce/system/smartmontools`

## Installation

### From Package Repository

```bash
pkg install system/monitoring/disk-agent
```

### Manual Installation

```bash
pkg install -g disk-agent-x.x.x.p5p system/monitoring/disk-agent
```

### Enable and Start Service

```bash
svcadm enable system/monitoring/disk-agent
svcs system/monitoring/disk-agent
tail -f /var/log/disk-agent/application.log
```

## Configuration

The service reads the configuration directory `/etc/disk-agent`: `app.config.yaml` and `auth.config.yaml`. Both files are preserved during package updates.

### API keys

```bash
pfexec su diskagent -c "/opt/disk-agent/disk-agent --config /etc/disk-agent key add homeassistant"
pfexec su diskagent -c "/opt/disk-agent/disk-agent --config /etc/disk-agent key list"
pfexec su diskagent -c "/opt/disk-agent/disk-agent --config /etc/disk-agent key remove 1"
```

### SSL Certificates

SSL certificates are generated during first startup if they do not exist:

- **Private Key**: `/etc/disk-agent/ssl/server.key`
- **Certificate**: `/etc/disk-agent/ssl/server.crt`

## API Access

Once running, the API is available at:

- **HTTP**: `http://localhost:9430`
- **HTTPS**: `https://localhost:9431`

## Service Management

```bash
svcadm enable system/monitoring/disk-agent
svcadm disable system/monitoring/disk-agent
svcadm restart system/monitoring/disk-agent
svcs -l system/monitoring/disk-agent
```

### Log Files

- **Service Log**: `/var/log/disk-agent/application.log`
- **SMF Log**: `/var/svc/log/system-monitoring-disk-agent:default.log`

## Build Process

The package is built automatically via GitHub Actions when a new release is created. The build process:

1. Builds the Go binary with the release version
2. Copies application files to the staging area
3. Creates the IPS package with proper permissions and ownership
4. Uploads the package to GitHub releases
5. Publishes to the package repository

## Troubleshooting

### Service Won't Start

1. Check SMF service status: `svcs -xv system/monitoring/disk-agent`
2. Check service logs: `tail -f /var/svc/log/system-monitoring-disk-agent:default.log`
3. Check application logs: `tail -f /var/log/disk-agent/application.log`
4. Verify configuration: `/etc/disk-agent/app.config.yaml`

## Uninstallation

```bash
svcadm disable system/monitoring/disk-agent
pkg uninstall system/monitoring/disk-agent
rm -rf /var/lib/disk-agent
rm -rf /var/log/disk-agent
```

## Support

- **GitHub**: https://github.com/MarkProminic/disk-agent
- **Issues**: https://github.com/MarkProminic/disk-agent/issues
