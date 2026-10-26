#!/usr/bin/bash
#
# CDDL HEADER START
#
# The contents of this file are subject to the terms of the
# Common Development and Distribution License, Version 1.0 only
# (the "License").  You may not use this file except in compliance
# with the License.
#
# You can obtain a copy of the license at usr/src/OPENSOLARIS.LICENSE
# or http://www.opensolaris.org/os/licensing.
# See the License for the specific language governing permissions
# and limitations under the License.
#
# When distributing Covered Code, include this CDDL HEADER in each
# file and include the License file at usr/src/OPENSOLARIS.LICENSE.
# If applicable, add the following below this CDDL HEADER, with the
# fields enclosed by brackets "[]" replaced with your own identifying
# information: Portions Copyright [yyyy] [name of copyright owner]
#
# CDDL HEADER END
#
#
# Copyright 2026 Makr91. All rights reserved.
# Use is subject to license terms.
#

set -e

logmsg() { echo "=== $*"; }
logcmd() { echo ">>> $*"; "$@"; }
logerr() { echo "ERROR: $*" >&2; }

SRCDIR="$(pwd)"
DESTDIR="${SRCDIR}/proto"
PROG=disk-agent
VER="${VERSION:-$(sed -n 's/.*Version = "\([^"]*\)".*/\1/p' internal/version/version.go)}"
PKG=system/monitoring/disk-agent

rm -rf "$DESTDIR"
mkdir -p "$DESTDIR"

build_app() {
    logmsg "Building Disk Agent"
    export CGO_ENABLED=0
    logcmd go build -trimpath -ldflags "-s -w -X github.com/MarkProminic/disk-agent/internal/version.Version=${VER}" -o "${PROG}" .
}

install_app() {
    pushd $DESTDIR >/dev/null

    logcmd mkdir -p opt/disk-agent
    pushd opt/disk-agent >/dev/null

    logmsg "Installing Disk Agent application files"
    logcmd cp $SRCDIR/${PROG} .
    logcmd cp $SRCDIR/LICENSE.md .

    logcmd cp $SRCDIR/packaging/omnios/startup.sh .
    logcmd chmod 755 startup.sh
    logcmd cp $SRCDIR/packaging/omnios/setup.sh .
    logcmd chmod 755 setup.sh
    logcmd cp $SRCDIR/packaging/omnios/trigger.sh .
    logcmd chmod 755 trigger.sh

    popd >/dev/null

    logmsg "Installing host event hooks"
    logcmd mkdir -p etc/sysevent/config etc/opt/ooce/smartmontools/smartd_warning.d
    logcmd cp $SRCDIR/packaging/omnios/disk-agent.sysevent.conf etc/sysevent/config/DiskAgent,sysevent.conf
    logcmd cp $SRCDIR/packaging/omnios/smartd_warning.d-disk-agent etc/opt/ooce/smartmontools/smartd_warning.d/disk-agent
    logcmd chmod 755 etc/opt/ooce/smartmontools/smartd_warning.d/disk-agent

    logmsg "Installing configuration files"
    logcmd mkdir -p etc/disk-agent
    logcmd cp $SRCDIR/packaging/omnios/app.config.yaml etc/disk-agent/app.config.yaml
    logcmd cp $SRCDIR/internal/config/seed/auth.config.yaml etc/disk-agent/auth.config.yaml

    logmsg "Installing RBAC fragments for the Disk Agent profile"
    logcmd mkdir -p etc/user_attr.d etc/security/prof_attr.d etc/security/exec_attr.d
    logcmd cp $SRCDIR/packaging/omnios/disk-agent.user_attr etc/user_attr.d/disk-agent
    logcmd cp $SRCDIR/packaging/omnios/disk-agent.prof_attr etc/security/prof_attr.d/disk-agent
    logcmd cp $SRCDIR/packaging/omnios/disk-agent.exec_attr etc/security/exec_attr.d/disk-agent

    logcmd mkdir -p etc/disk-agent/ssl
    logcmd mkdir -p var/lib/disk-agent
    logcmd mkdir -p var/log/disk-agent

    logmsg "Installing SMF manifest"
    logcmd mkdir -p lib/svc/manifest/system
    logcmd cp $SRCDIR/packaging/omnios/disk-agent-smf.xml lib/svc/manifest/system/disk-agent.xml
    logcmd cp $SRCDIR/packaging/omnios/disk-agent-setup-smf.xml lib/svc/manifest/system/disk-agent-setup.xml

    logmsg "Installing man pages"
    logcmd mkdir -p opt/ooce/share/man/man8 opt/ooce/share/man/man5
    logcmd cp $SRCDIR/packaging/DEBIAN/man/disk-agent.8 opt/ooce/share/man/man8/ || \
        logerr "--- copying main man page failed"
    logcmd cp $SRCDIR/packaging/DEBIAN/man/disk-agent.yaml.5 opt/ooce/share/man/man5/ || \
        logerr "--- copying config man page failed"

    popd >/dev/null
}

logmsg "Starting Disk Agent build process"
build_app
install_app

logmsg "Creating IPS package"
cd "$SRCDIR"
export VERSION="$VER"
sed "s/@VERSION@/${VERSION}/g" packaging/omnios/disk-agent.p5m > disk-agent.p5m.tmp
pkgsend generate proto | pkgfmt > disk-agent.p5m.generated
pkgmogrify -DVERSION="${VERSION}" disk-agent.p5m.tmp disk-agent.p5m.generated > disk-agent.p5m.final

TEMP_REPO="${SRCDIR}/temp-repo"
rm -rf "$TEMP_REPO"
pkgrepo create "$TEMP_REPO"
pkgrepo set -s "$TEMP_REPO" publisher/prefix=Makr91

pkgsend -s "file://${TEMP_REPO}" publish -d proto disk-agent.p5m.final

PACKAGE_FILE="disk-agent-${VERSION}.p5p"
pkgrecv -s "file://${TEMP_REPO}" -a -d "${PACKAGE_FILE}" "${PKG}"

rm -rf "$TEMP_REPO"

logmsg "Package build completed: ${PACKAGE_FILE}"
logmsg "Complete package ready for upload to GitHub artifacts"

# Vim hints
# vim:ts=4:sw=4:et:
