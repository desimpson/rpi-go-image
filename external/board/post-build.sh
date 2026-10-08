#!/bin/sh
# Runs after Buildroot has assembled the target rootfs (and after the overlay
# has been applied), before the filesystem image is created.
#
# $1 = TARGET_DIR
set -eu

TARGET_DIR="$1"

test -f "${TARGET_DIR}/etc/systemd/system/myapp.service" || {
    echo "post-build: /etc/systemd/system/myapp.service missing from rootfs" >&2
    exit 1
}

# Fail the build loudly if the binary was forgotten.
test -x "${TARGET_DIR}/usr/bin/myapp" || {
    echo "post-build: /usr/bin/myapp missing from rootfs; run 'make app' first" >&2
    exit 1
}

# Enable myapp.service. This is exactly what `systemctl enable myapp` does:
# create a symlink in multi-user.target.wants. Doing it here instead of
# shipping the symlink in git keeps the repo friendly to Windows checkouts.
mkdir -p "${TARGET_DIR}/etc/systemd/system/multi-user.target.wants"
ln -sf ../myapp.service \
    "${TARGET_DIR}/etc/systemd/system/multi-user.target.wants/myapp.service"

mkdir -p "${TARGET_DIR}/etc/systemd/system/sysinit.target.wants"
ln -sf ../usb-gadget.service \
    "${TARGET_DIR}/etc/systemd/system/sysinit.target.wants/usb-gadget.service"

# The Pi board script enables an HDMI login prompt, and systemd's default
# instance is tty1. This image has no login prompt: myapp is the USB serial
# console. Clear DefaultInstance before preset-all, or tty1 comes back.
rm -f \
    "${TARGET_DIR}/etc/systemd/system/getty.target.wants/getty@tty1.service" \
    "${TARGET_DIR}/etc/systemd/system/getty.target.wants/serial-getty@ttyAMA0.service" \
    "${TARGET_DIR}/etc/systemd/system/getty.target.wants/serial-getty@ttyGS0.service"
rm -f "${TARGET_DIR}/usr/lib/systemd/system/serial-getty@.service.d/buildroot-console.conf"
mkdir -p "${TARGET_DIR}/usr/lib/systemd/system/getty@.service.d"
printf '%s\n' '[Install]' 'DefaultInstance=' \
    > "${TARGET_DIR}/usr/lib/systemd/system/getty@.service.d/buildroot-console.conf"
mkdir -p "${TARGET_DIR}/etc/systemd/system-generators"
ln -sf /dev/null \
    "${TARGET_DIR}/etc/systemd/system-generators/systemd-getty-generator"

ln -sf /usr/lib/systemd/system/systemd-networkd.service \
    "${TARGET_DIR}/etc/systemd/system/multi-user.target.wants/systemd-networkd.service"
mkdir -p "${TARGET_DIR}/etc/systemd/system/sockets.target.wants"
ln -sf /usr/lib/systemd/system/systemd-networkd.socket \
    "${TARGET_DIR}/etc/systemd/system/sockets.target.wants/systemd-networkd.socket"

ln -sf /usr/lib/systemd/system/dropbear.service \
    "${TARGET_DIR}/etc/systemd/system/multi-user.target.wants/dropbear.service"
