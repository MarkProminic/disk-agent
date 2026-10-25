#!/bin/bash

set -e

export PATH="/opt/ooce/bin:/opt/ooce/sbin:/usr/bin:/usr/sbin:/sbin"
export CONFIG_DIR="${CONFIG_DIR:-/etc/disk-agent}"
export HOME="${HOME:-/var/lib/disk-agent}"

cd /opt/disk-agent

PIDFILE="/var/lib/disk-agent/disk-agent.pid"

mkdir -p /var/lib/disk-agent
mkdir -p /etc/disk-agent/ssl

chown -R diskagent:diskagent /var/lib/disk-agent
chown -R diskagent:diskagent /etc/disk-agent/ssl

chmod 700 /etc/disk-agent/ssl

if [ ! -x "/opt/disk-agent/disk-agent" ]; then
    echo "Error: Disk Agent binary not found at /opt/disk-agent/disk-agent" >&2
    exit 1
fi

if [ -f "$PIDFILE" ]; then
    if ! kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        echo "Removing stale PID file $PIDFILE"
        rm -f "$PIDFILE"
    else
        echo "Error: Disk Agent appears to be already running (PID $(cat "$PIDFILE"))" >&2
        exit 1
    fi
fi

echo "Starting Disk Agent..."
echo "Configuration: $CONFIG_DIR"

nohup /opt/disk-agent/disk-agent --config "$CONFIG_DIR" </dev/null >>/var/log/disk-agent/application.log 2>&1 &
AGENT_PID=$!

echo $AGENT_PID > "$PIDFILE"

sleep 2
if ! kill -0 $AGENT_PID 2>/dev/null; then
    echo "Error: Disk Agent failed to start" >&2
    rm -f "$PIDFILE"
    exit 1
fi

echo "Disk Agent started successfully with PID $AGENT_PID"

exit 0
