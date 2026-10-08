package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

const prompt = "myboard> "

const helpText = `commands:
  help
  version
  led status|on|off|blink
  exit`

const ledUsage = "led status|on|off|blink"

// command is one parsed console line.
type command interface {
	isCommand()
}

type emptyCommand struct{}
type helpCommand struct{}
type versionCommand struct{}
type exitCommand struct{}
type ledCommand struct{ action ledAction }
type unknownCommand struct{ name string }

func (emptyCommand) isCommand()   {}
func (helpCommand) isCommand()    {}
func (versionCommand) isCommand() {}
func (exitCommand) isCommand()    {}
func (ledCommand) isCommand()     {}
func (unknownCommand) isCommand() {}

// ledAction is what a led command does.
type ledAction int

const (
	ledStatus ledAction = iota
	ledOn
	ledOff
	ledBlink
)

// usageError is a known command with arguments that do not match its form.
type usageError struct {
	usage string
}

func (e usageError) Error() string {
	return "usage: " + e.usage
}

// parseLine converts one console line into a command.
// A blank line is an empty command. An unknown name is still a command;
// only a known name with bad arguments returns an error.
func parseLine(line string) (command, error) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return emptyCommand{}, nil
	}
	switch fields[0] {
	case "help":
		return oneWord("help", fields, helpCommand{})
	case "version":
		return oneWord("version", fields, versionCommand{})
	case "exit":
		return oneWord("exit", fields, exitCommand{})
	case "led":
		return parseLED(fields[1:])
	default:
		return unknownCommand{name: fields[0]}, nil
	}
}

func oneWord(name string, fields []string, cmd command) (command, error) {
	if len(fields) != 1 {
		return nil, usageError{usage: name}
	}
	return cmd, nil
}

func parseLED(args []string) (command, error) {
	if len(args) != 1 {
		return nil, usageError{usage: ledUsage}
	}
	switch args[0] {
	case "status":
		return ledCommand{action: ledStatus}, nil
	case "on":
		return ledCommand{action: ledOn}, nil
	case "off":
		return ledCommand{action: ledOff}, nil
	case "blink":
		return ledCommand{action: ledBlink}, nil
	default:
		return nil, usageError{usage: ledUsage}
	}
}

func banner(version string) string {
	return fmt.Sprintf("myboard %s\ntype help", version)
}

// console runs commands against the ACT LED.
// A nil LED still accepts every command and reports that the LED is absent.
type console struct {
	mu       sync.Mutex
	led      *led
	version  string
	blinking bool
	cancel   context.CancelFunc
}

func newConsole(l *led, version string) *console {
	return &console{led: l, version: version}
}

// close stops a blink started by a led command.
func (c *console) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopBlink()
}

// reply runs one command.
// endSession is true after exit, which also stops a blink. The process
// stays up; the caller prints the banner again for a new session.
func (c *console) reply(cmd command) (text string, endSession bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch cmd := cmd.(type) {
	case emptyCommand:
		return "", false, nil
	case helpCommand:
		return helpText, false, nil
	case versionCommand:
		return c.version, false, nil
	case exitCommand:
		c.stopBlink()
		return "session ended", true, nil
	case ledCommand:
		text, err = c.applyLED(cmd.action)
		return text, false, err
	case unknownCommand:
		return fmt.Sprintf("unknown command %q\ntype help", cmd.name), false, nil
	default:
		panic(fmt.Sprintf("unhandled command %T", cmd))
	}
}

func (c *console) applyLED(action ledAction) (string, error) {
	switch action {
	case ledStatus:
		return c.ledStatus(), nil
	case ledOn:
		return c.setLED(true)
	case ledOff:
		return c.setLED(false)
	case ledBlink:
		if c.led == nil {
			return "no LED", nil
		}
		c.startBlink()
		return "blinking", nil
	default:
		panic(fmt.Sprintf("unhandled LED action %d", action))
	}
}

func (c *console) setLED(on bool) (string, error) {
	c.stopBlink()
	if c.led == nil {
		return "no LED", nil
	}
	if err := c.led.set(on); err != nil {
		return "", err
	}
	if on {
		return "on", nil
	}
	return "off", nil
}

func (c *console) ledStatus() string {
	if c.led == nil {
		return "no LED"
	}
	if c.blinking {
		return "blinking"
	}
	if c.led.on {
		return "on"
	}
	return "off"
}

// startBlink toggles the LED twice a second until stopBlink.
// Caller holds c.mu. The goroutine takes the lock only around each write.
func (c *console) startBlink() {
	c.stopBlink()
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.blinking = true
	led := c.led
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.mu.Lock()
				if ctx.Err() != nil {
					c.mu.Unlock()
					return
				}
				err := led.toggle()
				c.mu.Unlock()
				if err != nil {
					log.Printf("LED: %v", err)
					return
				}
			}
		}
	}()
}

// stopBlink ends the blink goroutine. Caller holds c.mu.
func (c *console) stopBlink() {
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.blinking = false
}
