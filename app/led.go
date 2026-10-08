package main

import (
	"os"
	"path/filepath"
	"strings"
)

// led controls a Linux sysfs LED (e.g. the Pi's green ACT LED).
type led struct {
	dir     string
	on      bool
	restore string
}

// The ACT LED is called "ACT" on recent Raspberry Pi kernels and "led0" on
// older ones.
var defaultLEDDirs = []string{"/sys/class/leds/ACT", "/sys/class/leds/led0"}

// findLED returns the first LED directory that exists, or nil if none do.
func findLED(dirs ...string) *led {
	for _, d := range dirs {
		if _, err := os.Stat(filepath.Join(d, "brightness")); err == nil {
			return &led{dir: d}
		}
	}
	return nil
}

// takeOver stops the kernel's own trigger (mmc0 disk activity by default) so
// that we are the only thing driving the LED. A missing trigger file is not
// an error. When the file exists, its active trigger (the bracketed name,
// or mmc0 if none is marked) is kept for release.
func (l *led) takeOver() error {
	path := filepath.Join(l.dir, "trigger")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	name, ok := activeTrigger(string(data))
	if !ok {
		name = "mmc0"
	}
	l.restore = name
	return os.WriteFile(path, []byte("none"), 0o644)
}

// release hands the LED back to the trigger takeOver replaced.
func (l *led) release() error {
	if l.restore == "" {
		return nil
	}
	return os.WriteFile(filepath.Join(l.dir, "trigger"), []byte(l.restore), 0o644)
}

// activeTrigger returns the bracketed trigger name from a sysfs trigger file.
func activeTrigger(content string) (string, bool) {
	start := strings.IndexByte(content, '[')
	if start < 0 {
		return "", false
	}
	rest := content[start+1:]
	end := strings.IndexByte(rest, ']')
	if end <= 0 {
		return "", false
	}
	name := strings.TrimSpace(rest[:end])
	if name == "" {
		return "", false
	}
	return name, true
}

func (l *led) set(on bool) error {
	v := "0"
	if on {
		v = "1"
	}
	if err := os.WriteFile(filepath.Join(l.dir, "brightness"), []byte(v), 0o644); err != nil {
		return err
	}
	l.on = on
	return nil
}

func (l *led) toggle() error { return l.set(!l.on) }
