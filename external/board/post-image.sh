#!/bin/bash
# Builds sdcard.img from the files in BINARIES_DIR.
# See genimage.cfg.in for why this does not use the stock Raspberry Pi script.

set -e

BOARD_DIR="$(dirname "$0")"
GENIMAGE_CFG="${BINARIES_DIR}/genimage.cfg"
GENIMAGE_TMP="${BUILD_DIR}/genimage.tmp"

"${BOARD_DIR}/install-dwc2-overlay.sh"

# The firmware picks the DTB by board id. A Zero W will not fall back to the
# Zero DTB, and without a DTB the kernel never finds the SD card.
DTB_W="${BINARIES_DIR}/bcm2708-rpi-zero-w.dtb"
if [ ! -f "${DTB_W}" ]; then
	make -C "${BUILD_DIR}/linux-custom" ARCH=arm \
		CROSS_COMPILE="${HOST_DIR}/bin/arm-linux-" \
		broadcom/bcm2708-rpi-zero-w.dtb
	cp "${BUILD_DIR}/linux-custom/arch/arm/boot/dts/broadcom/bcm2708-rpi-zero-w.dtb" \
		"${DTB_W}"
fi

FILES=()

for i in "${BINARIES_DIR}"/*.dtb "${BINARIES_DIR}"/rpi-firmware/*; do
	FILES+=( "${i#${BINARIES_DIR}/}" )
done

KERNEL=$(sed -n 's/^kernel=//p' "${BINARIES_DIR}/rpi-firmware/config.txt")
FILES+=( "${KERNEL}" )

BOOT_FILES=$(printf '\\t\\t\\t"%s",\\n' "${FILES[@]}")
sed "s|#BOOT_FILES#|${BOOT_FILES}|" "${BOARD_DIR}/genimage.cfg.in" \
	> "${GENIMAGE_CFG}"

# Pass an empty rootpath. genimage makes a full copy of the given rootpath to
# ${GENIMAGE_TMP}/root so passing TARGET_DIR would be a waste of time and disk
# space. We don't rely on genimage to build the rootfs image, just to insert a
# pre-built one in the disk image.

trap 'rm -rf "${ROOTPATH_TMP}"' EXIT
ROOTPATH_TMP="$(mktemp -d)"

rm -rf "${GENIMAGE_TMP}"

genimage \
	--rootpath "${ROOTPATH_TMP}"   \
	--tmppath "${GENIMAGE_TMP}"    \
	--inputpath "${BINARIES_DIR}"  \
	--outputpath "${BINARIES_DIR}" \
	--config "${GENIMAGE_CFG}"

exit $?
