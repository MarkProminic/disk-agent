#!/bin/bash

set -e

CONF=/etc/opt/ooce/smartmontools/smartd.conf
LINE='DEVICESCAN -d removable -n standby -m <nomailer> -M exec /etc/opt/ooce/smartmontools/smartd_warning.sh'

if [ -f "$CONF" ] && ! grep -q '^DEVICESCAN.*-m ' "$CONF"; then
    if grep -q '^DEVICESCAN' "$CONF"; then
        awk -v line="$LINE" '/^DEVICESCAN/ { print line; next } { print }' "$CONF" > "$CONF.new"
        mv "$CONF.new" "$CONF"
    else
        echo "$LINE" >> "$CONF"
    fi
fi

if svcs -H -o state svc:/system/smartd:default 2>/dev/null | grep -q online; then
    svcadm refresh svc:/system/smartd:default
else
    svcadm enable svc:/system/smartd:default
fi

exit 0
