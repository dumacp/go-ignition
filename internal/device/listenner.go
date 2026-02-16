package device

import (
	"bufio"
	"context"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/dumacp/go-logs/pkg/logs"
	evdev "github.com/gvalkov/golang-evdev"
)

const (
	gpioMapFile = "/sys/kernel/debug/gpio"
)

// NewEventDevice connect with device serial
func NewEventDevice(path string) (*evdev.InputDevice, error) {
	dev, err := evdev.Open(path)
	if err != nil {
		return nil, err
	}
	return dev, nil
}

// EventUP up
type EventUP struct{}

// EventDown down
type EventDown struct{}

// Listen function to listen events
func Listen(ctx context.Context, dev *evdev.InputDevice) <-chan interface{} {

	ch := make(chan interface{})

	//first detection
	t0 := time.NewTimer(20 * time.Second)
	t1 := time.NewTimer(120 * time.Second)
	funcInitEvent := func() {

		if f, err := os.Open(gpioMapFile); err == nil {
			defer f.Close()
			re := regexp.MustCompile("(?i)ignition")
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Bytes()
				if re.Match(line) {
					reHIGH := regexp.MustCompile("hi")
					if reHIGH.Match(line) {
						logs.LogBuild.Println("ignition UP")
						select {
						case ch <- &EventUP{}:
						case <-time.After(3 * time.Second):
						case <-ctx.Done():
							return
						}
					} else {
						logs.LogBuild.Println("ignition DOWN")
						select {
						case ch <- &EventDown{}:
						case <-time.After(3 * time.Second):
						case <-ctx.Done():
							return
						}
					}
					break
				}
			}
		}
	}

	go func() {
		for {
			select {
			case <-t0.C:
				funcInitEvent()
			case <-t1.C:
				funcInitEvent()
			case <-ctx.Done():
				t0.Stop()
				t1.Stop()
				return
			}
		}
	}()

	go func() {
		defer close(ch)
		// var eventMem *evdev.InputEvent
		for {
			log.Println("leyendo!")
			iv, err := dev.ReadOne()
			if err != nil {
				continue
			}
			logs.LogBuild.Printf("event: %v", iv)
			switch iv.Code {
			case evdev.KEY_WAKEUP:
				if iv.Value == 0 {
					logs.LogBuild.Println("ignition Event UP")
					select {
					case ch <- &EventUP{}:
					case <-time.After(3 * time.Second):
					case <-ctx.Done():
						return
					}
				} else {
					logs.LogBuild.Println("ignition Event DOWN")
					select {
					case ch <- &EventDown{}:
					case <-time.After(3 * time.Second):
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return ch
}
