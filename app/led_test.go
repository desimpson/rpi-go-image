package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindLEDMissing(t *testing.T) {
	if findLED(filepath.Join(t.TempDir(), "nope")) != nil {
		t.Fatal("expected nil for missing LED")
	}
}

func TestLEDToggle(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "brightness"), []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}

	l := findLED("/does/not/exist", dir)
	if l == nil {
		t.Fatal("LED not found")
	}
	l.takeOver()

	read := func() string {
		b, err := os.ReadFile(filepath.Join(dir, "brightness"))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	if err := l.toggle(); err != nil || read() != "1" {
		t.Fatalf("after first toggle: err=%v brightness=%q", err, read())
	}
	if err := l.toggle(); err != nil || read() != "0" {
		t.Fatalf("after second toggle: err=%v brightness=%q", err, read())
	}
}

func TestFindLEDSkipsDirWithoutBrightness(t *testing.T) {
	// Arrange

	root := t.TempDir()
	empty := filepath.Join(root, "empty")
	good := filepath.Join(root, "good")
	if err := os.Mkdir(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(good, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(good, "brightness"), []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Act

	l := findLED(empty, good)

	// Assert

	if l == nil || l.dir != good {
		t.Fatalf("found %#v, want %s", l, good)
	}
}

func TestTakeOverPreservesTrigger(t *testing.T) {
	// Arrange

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "brightness"), []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "trigger"), []byte("none timer [mmc0] heartbeat\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	l := findLED(dir)
	if l == nil {
		t.Fatal("LED not found")
	}

	// Act

	err := l.takeOver()

	// Assert

	if err != nil {
		t.Fatal(err)
	}
	if l.restore != "mmc0" {
		t.Fatalf("restore=%q, want mmc0", l.restore)
	}
	got, err := os.ReadFile(filepath.Join(dir, "trigger"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "none" {
		t.Fatalf("trigger=%q, want none", got)
	}
}

func TestTakeOverDefaultsToMMC0(t *testing.T) {
	// Arrange

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "brightness"), []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "trigger"), []byte("timer heartbeat\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	l := findLED(dir)
	if l == nil {
		t.Fatal("LED not found")
	}

	// Act

	err := l.takeOver()

	// Assert

	if err != nil {
		t.Fatal(err)
	}
	if l.restore != "mmc0" {
		t.Fatalf("restore=%q, want mmc0", l.restore)
	}
}

func TestTakeOverMissingTrigger(t *testing.T) {
	// Arrange

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "brightness"), []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}
	l := findLED(dir)
	if l == nil {
		t.Fatal("LED not found")
	}

	// Act

	err := l.takeOver()

	// Assert

	if err != nil {
		t.Fatal(err)
	}
	if l.restore != "" {
		t.Fatalf("restore=%q, want empty", l.restore)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "trigger")); !os.IsNotExist(statErr) {
		t.Fatalf("trigger file appeared: %v", statErr)
	}
}

func TestReleaseRestoresTrigger(t *testing.T) {
	// Arrange

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "brightness"), []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "trigger"), []byte("[actpwr] mmc0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	l := findLED(dir)
	if l == nil {
		t.Fatal("LED not found")
	}
	if err := l.takeOver(); err != nil {
		t.Fatal(err)
	}

	// Act

	err := l.release()

	// Assert

	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "trigger"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "actpwr" {
		t.Fatalf("trigger=%q, want actpwr", got)
	}
}

func TestToggleReadOnly(t *testing.T) {
	// Arrange

	// /dev/full rejects writes for every user, including root. A mode-0444
	// file does not, because root ignores permission bits.
	dir := t.TempDir()
	if err := os.Symlink("/dev/full", filepath.Join(dir, "brightness")); err != nil {
		t.Fatal(err)
	}
	l := findLED(dir)
	if l == nil {
		t.Fatal("LED not found")
	}

	// Act

	err := l.toggle()

	// Assert

	if err == nil {
		t.Fatal("expected write error")
	}
}
