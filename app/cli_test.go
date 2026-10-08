package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseHelp(t *testing.T) {
	// Arrange

	line := "help"

	// Act

	cmd, err := parseLine(line)

	// Assert

	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cmd.(helpCommand); !ok {
		t.Fatalf("command = %T, want helpCommand", cmd)
	}
}

func TestParseBlank(t *testing.T) {
	// Arrange

	line := "   "

	// Act

	cmd, err := parseLine(line)

	// Assert

	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cmd.(emptyCommand); !ok {
		t.Fatalf("command = %T, want emptyCommand", cmd)
	}
}

func TestParseLEDBlink(t *testing.T) {
	// Arrange

	line := "led blink"

	// Act

	cmd, err := parseLine(line)

	// Assert

	if err != nil {
		t.Fatal(err)
	}
	got, ok := cmd.(ledCommand)
	if !ok || got.action != ledBlink {
		t.Fatalf("command = %#v, want led blink", cmd)
	}
}

func TestParseLEDUsage(t *testing.T) {
	// Arrange

	line := "led disco"

	// Act

	_, err := parseLine(line)

	// Assert

	var usage usageError
	if !errors.As(err, &usage) {
		t.Fatalf("err = %v, want usageError", err)
	}
	if usage.usage != ledUsage {
		t.Fatalf("usage = %q, want %q", usage.usage, ledUsage)
	}
}

func TestParseUnknown(t *testing.T) {
	// Arrange

	line := "reboot now"

	// Act

	cmd, err := parseLine(line)

	// Assert

	if err != nil {
		t.Fatal(err)
	}
	got, ok := cmd.(unknownCommand)
	if !ok || got.name != "reboot" {
		t.Fatalf("command = %#v, want unknown reboot", cmd)
	}
}

func TestReplyLEDOn(t *testing.T) {
	// Arrange

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "brightness"), []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}
	l := findLED(dir)
	if l == nil {
		t.Fatal("LED not found")
	}
	cons := newConsole(l, "test")

	// Act

	text, endSession, err := cons.reply(ledCommand{action: ledOn})

	// Assert

	if err != nil {
		t.Fatal(err)
	}
	if endSession {
		t.Fatal("led on ended the session")
	}
	if text != "on" {
		t.Fatalf("text = %q, want on", text)
	}
	got, err := os.ReadFile(filepath.Join(dir, "brightness"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "1" {
		t.Fatalf("brightness = %q, want 1", got)
	}
}

func TestReplyNoLED(t *testing.T) {
	// Arrange

	cons := newConsole(nil, "test")

	// Act

	text, _, err := cons.reply(ledCommand{action: ledStatus})

	// Assert

	if err != nil {
		t.Fatal(err)
	}
	if text != "no LED" {
		t.Fatalf("text = %q, want no LED", text)
	}
}

func TestReplyExitStopsBlink(t *testing.T) {
	// Arrange

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "brightness"), []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}
	l := findLED(dir)
	if l == nil {
		t.Fatal("LED not found")
	}
	cons := newConsole(l, "test")
	if _, _, err := cons.reply(ledCommand{action: ledBlink}); err != nil {
		t.Fatal(err)
	}

	// Act

	text, endSession, err := cons.reply(exitCommand{})

	// Assert

	if err != nil {
		t.Fatal(err)
	}
	if !endSession {
		t.Fatal("exit did not end the session")
	}
	if text != "session ended" {
		t.Fatalf("text = %q, want session ended", text)
	}
	if cons.blinking {
		t.Fatal("blink still running")
	}
}

func TestServeExitPrintsBannerAgain(t *testing.T) {
	// Arrange

	cons := newConsole(nil, "test")
	in := strings.NewReader("exit\n")
	var out bytes.Buffer

	// Act

	err := serve(context.Background(), in, &out, cons)

	// Assert

	if err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Count(got, "myboard dev") != 1 {
		t.Fatalf("banner count wrong:\n%s", got)
	}
	if !strings.Contains(got, "session ended") {
		t.Fatalf("missing session ended:\n%s", got)
	}
	if !strings.HasSuffix(got, prompt) {
		t.Fatalf("output = %q, want a prompt at the end", got)
	}
}
