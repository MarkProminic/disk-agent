#!/bin/sh
exec /usr/lib/disk-agent/trigger.sh zfs "$ZEVENT_SUBCLASS" "$ZEVENT_VDEV_PATH"
