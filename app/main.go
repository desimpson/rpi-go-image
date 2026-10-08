// Command myapp is a demo serial console for a Raspberry Pi Zero image.
//
// systemd attaches it to the USB gadget serial port at boot. There is no
// login prompt: this program is the console. Its stderr is the journal.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

	dev := findLED(defaultLEDDirs...)
	if dev != nil {
		if err := dev.takeOver(); err != nil {
			log.Printf("LED trigger: %v", err)
		}
		defer func() {
			if err := dev.release(); err != nil {
				log.Printf("LED restore: %v", err)
			}
		}()
	} else {
		log.Printf("no ACT LED found")
	}

	// A blocked read does not notice SIGTERM until stdin is closed.
	go func() {
		<-ctx.Done()
		os.Stdin.Close()
	}()

	cons := newConsole(dev, version)
	defer cons.close()

	writeBanner(os.Stdout)
	if err := serve(ctx, os.Stdin, os.Stdout, cons); err != nil {
		log.Printf("console: %v", err)
	}
	log.Printf("shutting down")
}

func writeBanner(w io.Writer) {
	fmt.Fprintln(w, banner(version))
}

// serve reads lines until the console closes, the context is cancelled,
// or a read fails. exit starts a new session in the same process.
func serve(ctx context.Context, in io.Reader, out io.Writer, cons *console) error {
	lines := bufio.NewReader(in)
	for {
		if ctx.Err() != nil {
			return nil
		}
		fmt.Fprint(out, prompt)
		line, err := lines.ReadString('\n')
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		cmd, err := parseLine(strings.TrimRight(line, "\r\n"))
		if err != nil {
			fmt.Fprintln(out, err.Error())
			continue
		}
		text, endSession, err := cons.reply(cmd)
		if err != nil {
			log.Printf("command: %v", err)
			fmt.Fprintln(out, err.Error())
			continue
		}
		if text != "" {
			fmt.Fprintln(out, text)
		}
		if endSession {
			writeBanner(out)
		}
	}
}
