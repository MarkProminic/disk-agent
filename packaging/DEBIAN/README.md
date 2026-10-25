# Building Disk Agent Debian Packages

Production-ready Debian package build for the Disk Agent, with automated CI/CD via Release Please (`.github/workflows/prod-build.yml`, `build-linux` job) and publication to the STARTcloud Debian repository (`publish-debian` job).

## What the package installs

- `/usr/bin/disk-agent` - the single static binary
- `/usr/share/disk-agent/config/{app,auth}.config.yaml` - the seed configuration files (from `internal/config/seed/`); `postinst` copies each to `/etc/disk-agent/<name>.config.yaml` when absent and creates `/etc/disk-agent/setup.token`
- `/etc/systemd/system/disk-agent.service` - service unit
- `/etc/sudoers.d/disk-agent` - lets the `disk-agent` user run `smartctl` and `zpool` as root
- `/usr/share/man/man8/disk-agent.8.gz`, `/usr/share/man/man5/disk-agent.yaml.5.gz` - manual pages

The service runs as the `disk-agent` system user with `--config /etc/disk-agent`, logs to the journal and `/var/log/disk-agent/`.

## Manual build

```bash
export VERSION=0.1.0 ARCH=amd64 PKG=disk-agent
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X github.com/MarkProminic/disk-agent/internal/version.Version=${VERSION}" -o bin/disk-agent .

STAGE="${PKG}_${VERSION}_${ARCH}"
mkdir -p "$STAGE"/usr/bin \
         "$STAGE"/etc/disk-agent \
         "$STAGE"/etc/systemd/system \
         "$STAGE"/etc/sudoers.d \
         "$STAGE"/usr/share/man/man8 \
         "$STAGE"/usr/share/man/man5 \
         "$STAGE"/usr/share/disk-agent/config \
         "$STAGE"/DEBIAN
cp bin/disk-agent "$STAGE/usr/bin/"
cp internal/config/seed/*.config.yaml "$STAGE/usr/share/disk-agent/config/"
cp packaging/DEBIAN/systemd/disk-agent.service "$STAGE/etc/systemd/system/"
cp packaging/DEBIAN/sudoers/disk-agent "$STAGE/etc/sudoers.d/disk-agent"
gzip -9 -c packaging/DEBIAN/man/disk-agent.8 > "$STAGE/usr/share/man/man8/disk-agent.8.gz"
gzip -9 -c packaging/DEBIAN/man/disk-agent.yaml.5 > "$STAGE/usr/share/man/man5/disk-agent.yaml.5.gz"
cp packaging/DEBIAN/postinst packaging/DEBIAN/prerm packaging/DEBIAN/postrm "$STAGE/DEBIAN/"

cat > "$STAGE/DEBIAN/control" << EOF
Package: disk-agent
Version: ${VERSION}
Section: admin
Priority: optional
Architecture: ${ARCH}
Depends: adduser, sudo, smartmontools, zfsutils-linux
Maintainer: Makr91 <makr91@users.noreply.github.com>
Description: Disk Agent - SMART and ZFS pool state for Home Assistant
 Single-binary host agent that serves the SMART state of every disk
 and the state of every ZFS pool as JSON over HTTP for the
 ha_disk_monitor Home Assistant integration to poll.
Homepage: https://github.com/MarkProminic/disk-agent
EOF

find "$STAGE" -type d -exec chmod 755 {} \;
find "$STAGE" -type f -exec chmod 644 {} \;
chmod 755 "$STAGE/usr/bin/disk-agent" "$STAGE/DEBIAN"/postinst "$STAGE/DEBIAN"/prerm "$STAGE/DEBIAN"/postrm
chmod 440 "$STAGE/etc/sudoers.d/disk-agent"

dpkg-deb --build --root-owner-group "$STAGE" "${STAGE}.deb"
```

## Service management

```bash
sudo systemctl enable --now disk-agent
sudo systemctl status disk-agent
sudo journalctl -fu disk-agent
```

## Uninstall

```bash
sudo apt remove disk-agent
sudo apt purge disk-agent
```
