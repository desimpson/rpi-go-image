#!/bin/sh
# Copies dwc2.dtbo onto the boot partition inputs before genimage runs.
# The Pi bootloader loads it from overlays/dwc2.dtbo.
set -eu

src=$(echo "${BUILD_DIR}"/rpi-firmware-*/boot/overlays/dwc2.dtbo)
if [ ! -f "${src}" ]; then
	echo "install-dwc2-overlay: ${src} not found" >&2
	exit 1
fi

mkdir -p "${BINARIES_DIR}/rpi-firmware/overlays"
cp "${src}" "${BINARIES_DIR}/rpi-firmware/overlays/dwc2.dtbo"
