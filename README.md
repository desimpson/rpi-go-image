# rpi-go-image

A minimal [Buildroot](https://buildroot.org) Linux image for the **Raspberry Pi Zero (v1 / ARMv6)**
that boots straight into a Go application running as a **systemd** service.
GitHub Actions builds the SD card image and publishes it as a download.

The demo app (`app/`) is the console. It starts on the USB serial port at boot, with no login prompt.
From that prompt you can turn the green ACT LED on, off, or blink it.

## Get an image

**From a release:** push a tag and download the image from the GitHub Release page. Only the newest release keeps the `.img.xz` and `.sha256`; older releases keep their notes and git tags.

```sh
git tag v0.1.0 && git push --tags
```

**From `main`:** the latest build is a workflow artifact named `sdcard-image`. A newer `main` build replaces it. Pull requests build the image and do not upload it. A manual run uploads one artifact, same as `main`. Its "clean" input ignores the download and ccache caches.

**Flash it:** Raspberry Pi Imager → *Choose OS* → *Use custom* → select `myboard-rpi0-<version>.img.xz`
(balenaEtcher also takes `.xz` directly). Don't use Imager's OS customisation (Wi-Fi/SSH/user) options;
they only apply to Raspberry Pi OS images.

**Boot it:** plug a data cable into the micro-USB port marked **USB** (not **PWR**). After the
Pi Zero boots (~10-20 s) the computer sees `/dev/ttyACM0`. Open it at 115200 baud and you are
already at the `myboard>` prompt.

```sh
picocom -b 115200 /dev/ttyACM0
```

Leave picocom with **Ctrl+A** then **Ctrl+X**.

```
myboard <version>
type help
myboard> help
myboard> led blink
myboard> led off
myboard> exit
```

`exit` ends the session and prints the banner again. It does not stop the service. There is no
login prompt on the serial port, HDMI, or the UART header.

## Logging in

SSH is a normal root shell, separate from the serial CLI. The password is `changeme`
(set in `external/configs/myboard_defconfig`; change it).

| Console | Where |
|---|---|
| USB serial | the demo CLI, on the Pi's **USB** port, `/dev/ttyACM0` on the computer |
| SSH | `ssh root@192.168.7.2` over the USB Ethernet interface |

The USB port is one device with two functions: a serial port (`ttyGS0` on the Pi) and an Ethernet interface (`usb0`, address `192.168.7.2/24`). The Pi offers DHCP on that link and does not advertise itself as a default router. The computer should receive `192.168.7.1`.

```sh
systemctl status myapp
journalctl -u myapp -f
```

## Build locally (Linux)

```sh
sudo apt-get install build-essential libncurses-dev file cpio unzip rsync bc wget git golang
make dist          # -> dist/myboard-rpi0-<version>.img.xz
```

Other targets: `make test`, `make app`, `make menuconfig`, `make savedefconfig`, `make clean`.
The first build downloads and compiles the kernel and systemd (roughly 20-40 minutes);
rebuilds after changing only the Go app take a minute or two.

## How it works

| Path | Purpose |
|---|---|
| `app/` | The Go program. Cross-compiled with `GOARCH=arm GOARM=6 CGO_ENABLED=0` (static binary). |
| `BUILDROOT_VERSION` | Pinned Buildroot release (LTS). Cloned on demand; not vendored or a submodule. |
| `external/configs/myboard_defconfig` | Buildroot's stock `raspberrypi0_defconfig` plus systemd, hostname, root password, overlay, ccache, and Dropbear. |
| `external/board/config.txt` | Boot config. Loads the `dwc2` USB peripheral overlay. |
| `external/board/overlay/` | Files copied into the rootfs: `myapp`, the USB gadget service, and the `usb0` network config. |
| `external/board/post-build.sh` | Enables `myapp`, the USB gadget, networkd, and Dropbear, and disables getty so the USB serial port is the demo CLI. Fails the build if the binary or unit file is missing. |
| `external/board/post-image.sh` | Assembles the SD card image. The boot partition is FAT32 and starts at 4 MiB, which is the layout the Pi boot ROM reads. |
| `.github/workflows/image.yml` | test → build image (downloads and ccache are cached) → keep the latest `main` artifact → publish a release on `v*` tags and drop older release images. Pull requests do not upload the image. |

## Customising

- **Change the app:** edit `app/`, keep the binary name `myapp` (or rename it in the Makefile, the unit file and `post-build.sh`).
- **Change the config:** `make menuconfig`, then `make savedefconfig` to write it back to the repo and commit.
- **Pi Zero W:** the image includes `bcm2708-rpi-zero-w.dtb`, so a Zero W boots it. Wi-Fi firmware is not included; start from Buildroot's `raspberrypi0w_defconfig` for that.
- **Upgrade Buildroot:** change `BUILDROOT_VERSION` to a newer tag (the `.02` releases are LTS). The next build deletes the cloned `buildroot/` tree and `output/`, then clones that tag.

## Next steps (not included)

- Read-only rootfs, A/B updates (RAUC/SWUpdate), or a hardware watchdog for appliance-grade robustness.
- Move the Go build into Buildroot as a `golang-package` so Buildroot rebuilds it itself.
