#!/bin/sh
SOCKET="${DISK_AGENT_SOCKET:-/etc/disk-agent/trigger.sock}"
exec curl -s -o /dev/null --unix-socket "$SOCKET" -X POST \
    --data-urlencode "kind=$1" \
    --data-urlencode "subclass=$2" \
    --data-urlencode "device=$3" \
    http://disk-agent/api/trigger
