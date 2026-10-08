// Command myapp is a tiny demo service for a Raspberry Pi Zero image.
//
// It blinks the green ACT LED once a second (so you can see it is alive
// without any screen or serial cable) and writes a heartbeat line to stdout,
// which systemd captures in the journal.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// version is stamped at build time: -ldflags "-X main.version=v1.2.3"
var version = "dev"

func main() {
	// journald adds its own timestamps. The Pi Zero has no battery-backed
	// clock, so wall-clock time is meaningless right after boot anyway.
	log.SetFlags(0)
	log.Printf("myapp %s starting (pid %d)", version, os.Getpid())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	status := findLED(defaultLEDDirs...)
	if status != nil {
		// Keep a stable pointer: the loop clears status after a write error,
		// and shutdown still has to restore the kernel trigger.
		ledDev := status
		if err := status.takeOver(); err != nil {
			log.Printf("LED trigger: %v", err)
		}
		log.Printf("blinking LED %s", status.dir)
		defer func() {
			if err := ledDev.release(); err != nil {
				log.Printf("LED restore: %v", err)
			}
		}()
	} else {
		log.Printf("no ACT LED found, running without blinking")
	}

	start := time.Now()
	blink := time.NewTicker(500 * time.Millisecond)
	beat := time.NewTicker(10 * time.Second)
	defer blink.Stop()
	defer beat.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("shutting down after %s", time.Since(start).Round(time.Second))
			return
		case <-blink.C:
			if status != nil {
				if err := status.toggle(); err != nil {
					log.Printf("LED error: %v", err)
					status = nil
				}
			}
		case <-beat.C:
			log.Printf("heartbeat: up %s", time.Since(start).Round(time.Second))
		}
	}
}
