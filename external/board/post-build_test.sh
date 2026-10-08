#!/bin/sh
# Exercises post-build.sh without Buildroot.
set -eu

script_dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
post_build="${script_dir}/post-build.sh"

fail() {
    echo "post-build_test: $*" >&2
    exit 1
}

prepare_root() {
    dir=$1
    mkdir -p "${dir}/usr/bin" "${dir}/etc/systemd/system"
    printf '#!/bin/sh\n' > "${dir}/usr/bin/myapp"
    chmod +x "${dir}/usr/bin/myapp"
    printf '%s\n' '[Service]' > "${dir}/etc/systemd/system/myapp.service"
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# Arrange

missing_bin="${tmp}/missing-bin"
mkdir -p "${missing_bin}/etc/systemd/system"
printf '%s\n' '[Service]' > "${missing_bin}/etc/systemd/system/myapp.service"

# Act

set +e
"${post_build}" "${missing_bin}" >/dev/null 2>&1
status=$?
set -e

# Assert

if [ "$status" -eq 0 ]; then
    fail "missing binary should fail the build"
fi

# Arrange

missing_unit="${tmp}/missing-unit"
mkdir -p "${missing_unit}/usr/bin"
printf '#!/bin/sh\n' > "${missing_unit}/usr/bin/myapp"
chmod +x "${missing_unit}/usr/bin/myapp"

# Act

set +e
"${post_build}" "${missing_unit}" >/dev/null 2>&1
status=$?
set -e

# Assert

if [ "$status" -eq 0 ]; then
    fail "missing unit file should fail the build"
fi

# Arrange

ok="${tmp}/ok"
prepare_root "${ok}"
mkdir -p "${ok}/etc/systemd/system/getty.target.wants"
ln -s /lib/systemd/system/getty@.service \
    "${ok}/etc/systemd/system/getty.target.wants/getty@tty1.service"
link="${ok}/etc/systemd/system/multi-user.target.wants/myapp.service"

# Act

"${post_build}" "${ok}"

# Assert

if [ ! -L "$link" ]; then
    fail "enable symlink was not created"
fi
target=$(readlink "$link")
if [ "$target" != "../myapp.service" ]; then
    fail "symlink target is ${target}"
fi
gadget="${ok}/etc/systemd/system/sysinit.target.wants/usb-gadget.service"
if [ "$(readlink "$gadget")" != "../usb-gadget.service" ]; then
    fail "usb-gadget service was not enabled"
fi
if [ -e "${ok}/etc/systemd/system/getty.target.wants/getty@tty1.service" ]; then
    fail "HDMI getty was left enabled"
fi
if [ -e "${ok}/etc/systemd/system/getty.target.wants/serial-getty@ttyGS0.service" ]; then
    fail "USB serial getty was enabled"
fi
generator="${ok}/etc/systemd/system-generators/systemd-getty-generator"
if [ "$(readlink "$generator")" != "/dev/null" ]; then
    fail "getty generator was not masked"
fi
dropin="${ok}/usr/lib/systemd/system/getty@.service.d/buildroot-console.conf"
if ! grep -qx 'DefaultInstance=' "$dropin"; then
    fail "getty default instance was not cleared"
fi
networkd="${ok}/etc/systemd/system/multi-user.target.wants/systemd-networkd.service"
if [ "$(readlink "$networkd")" != "/usr/lib/systemd/system/systemd-networkd.service" ]; then
    fail "systemd-networkd was not enabled"
fi
dropbear="${ok}/etc/systemd/system/multi-user.target.wants/dropbear.service"
if [ "$(readlink "$dropbear")" != "/usr/lib/systemd/system/dropbear.service" ]; then
    fail "dropbear was not enabled"
fi

# Arrange

again="${tmp}/again"
prepare_root "${again}"
again_link="${again}/etc/systemd/system/multi-user.target.wants/myapp.service"
"${post_build}" "${again}"

# Act

"${post_build}" "${again}"

# Assert

if [ ! -L "$again_link" ]; then
    fail "enable symlink disappeared on the second run"
fi
target=$(readlink "$again_link")
if [ "$target" != "../myapp.service" ]; then
    fail "symlink target changed to ${target}"
fi
