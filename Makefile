# Local build entry point. CI runs exactly the same targets.
#
#   make app        cross-compile the Go app into the rootfs overlay
#   make image      build the full SD card image (first run: ~20-40 min)
#   make dist       compress it into dist/*.img.xz (+ sha256)
#   make menuconfig tweak the Buildroot config interactively
#   make savedefconfig   write menuconfig changes back into the repo
#   make clean

BUILDROOT_VERSION := $(shell cat BUILDROOT_VERSION)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

BR_DIR  := $(CURDIR)/buildroot
OUT     := $(CURDIR)/output
EXT     := $(CURDIR)/external
APP_BIN := $(EXT)/board/overlay/usr/bin/myapp
# -j stays on this command line. external/board/buildroot-make.patch keeps
# Buildroot's MAKE variable as a bare program path so OpenSSL can quote it.
NPROC   := $(shell nproc 2>/dev/null || echo 1)
BR_MAKE := $(MAKE) -j$(NPROC) -C $(BR_DIR) BR2_EXTERNAL=$(EXT) O=$(OUT)

.PHONY: all app test image dist menuconfig savedefconfig clean distclean
all: image

test:
	cd app && go vet ./... && go test ./...
	$(CURDIR)/external/board/post-build_test.sh

# Pi Zero (v1 and W) is ARMv6 with hardware float: GOARM=6.
app:
	mkdir -p $(dir $(APP_BIN))
	cd app && CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 \
	    go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o $(APP_BIN) .

# BUILDROOT_VERSION is newer than an existing tree after a pin bump, so
# this reclones and drops output/ from the previous release.
$(BR_DIR)/Makefile: BUILDROOT_VERSION
	rm -rf $(BR_DIR) $(OUT)
	git clone --depth 1 --branch $(BUILDROOT_VERSION) \
	    https://github.com/buildroot/buildroot.git $(BR_DIR)
	patch -d $(BR_DIR) -p1 < $(EXT)/board/buildroot-make.patch

$(OUT)/.config: $(BR_DIR)/Makefile $(EXT)/configs/myboard_defconfig
	$(BR_MAKE) myboard_defconfig

image: app $(OUT)/.config
	$(BR_MAKE)

dist: image
	mkdir -p dist
	xz -T0 -9 -c $(OUT)/images/sdcard.img > dist/myboard-rpi0-$(VERSION).img.xz
	cd dist && sha256sum myboard-rpi0-$(VERSION).img.xz > myboard-rpi0-$(VERSION).img.xz.sha256
	@ls -lh dist

menuconfig: $(OUT)/.config
	$(BR_MAKE) menuconfig

savedefconfig: $(OUT)/.config
	$(BR_MAKE) savedefconfig BR2_DEFCONFIG=$(EXT)/configs/myboard_defconfig

clean:
	rm -rf $(OUT) dist $(APP_BIN)

distclean: clean
	rm -rf $(BR_DIR)
