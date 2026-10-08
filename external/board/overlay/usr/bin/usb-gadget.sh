#!/bin/sh
# Loads the USB OTG gadget after the dwc2 controller has registered.
# g_cdc presents one USB device with a serial port and an Ethernet interface.
set -eu

modprobe dwc2

n=0
while :; do
	set -- /sys/class/udc/*
	if [ -e "$1" ]; then
		break
	fi
	n=$((n + 1))
	if [ "$n" -ge 20 ]; then
		echo "usb-gadget: dwc2 did not register a UDC" >&2
		exit 1
	fi
	sleep 1
done

modprobe g_cdc
