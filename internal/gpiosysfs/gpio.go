package gpiosysfs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// Chip is the information of a GPIO controller chip.
type Chip struct {
	Base  int    // The first GPIO managed by this chip.
	Label string // The Label fo this chip. Provided for diagnostics (not always unique)
	Ngpio int    // How many GPIOs this chip manges. The GPIOs managed by this chip are in the range of Base to Base + ngpio - 1.
}

// Pin is a GPIO pin.
type Pin struct {
	n                                 int
	value, direction, edge, activeLow *gpioFile
}

// OpenPin opens the GPIO pin #n for IO.
func OpenPin(n int) (pin *Pin, err error) {
	dir := fmt.Sprintf("/sys/class/gpio/gpio%d", n)
	fi, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			err = fmt.Errorf("failed to open pin #%v: %w", n, err)
			return
		}
		const exportPath = "/sys/class/gpio/export"
		err = writeExisting(exportPath, strconv.Itoa(n))
		if err != nil {
			err = fmt.Errorf("failed to open pin #%v: %w", n, err)
			return
		}
	} else if !fi.IsDir() {
		err = fmt.Errorf("failed to open pin #%v: %v is not a dir", n, dir)
		return
	}

	pin = &Pin{n: n,
		direction: &gpioFile{Path: filepath.Join(dir, "direction")},
		value:     &gpioFile{Path: filepath.Join(dir, "value")},
		edge:      &gpioFile{Path: filepath.Join(dir, "edge")},
		activeLow: &gpioFile{Path: filepath.Join(dir, "active_low")},
	}
	return
}

// Close closes the pin.
func (pin *Pin) Close() error {
	if err := writeExisting("/sys/class/gpio/unexport", strconv.Itoa(pin.n)); err != nil {
		return fmt.Errorf("failed to close pin #%v: %w", pin.n, err)
	}

	if pin.value != nil {
		if err := pin.value.Close(); err != nil {
			return fmt.Errorf("failed to close pin #%v: %w", pin.n, err)
		}
	}
	if pin.direction != nil {
		if err := pin.direction.Close(); err != nil {
			return fmt.Errorf("failed to close pin #%v: %w", pin.n, err)
		}
	}
	if pin.edge != nil {
		if err := pin.edge.Close(); err != nil {
			return fmt.Errorf("failed to close pin #%v: %w", pin.n, err)
		}
	}
	if pin.activeLow != nil {
		if err := pin.activeLow.Close(); err != nil {
			return fmt.Errorf("failed to close pin #%v: %w", pin.n, err)
		}
	}
	return nil
}

// Direction is the IO direction.
type Direction string

// Available directions.
const (
	In      Direction = "in"   // RW. The pin is configured as input.
	Out     Direction = "out"  // RW. The pin is configured as output, usually initialized to low.
	OutLow  Direction = "low"  // W. Configure the pin as output and initialize it to low.
	OutHigh Direction = "high" // W. Configure the pin as output and initialize it to high.
)

// SetDirection sets the IO direction of the pin.
// pin.SetDirection(Out) may fail if Edge is not None.
func (pin *Pin) SetDirection(direction Direction) error {
	if _, err := pin.direction.WriteAt0([]byte(direction)); err != nil {
		return wrapPinError(pin, "set direction", err)
	}
	return nil
}

// Must be greater or equal to the max length of Edge and Direction constants.
const strBufLen = 16

// Direction returns the IO direction of the pin.
// The return values is In or Out.
func (pin *Pin) Direction() (Direction, error) {
	var buf [strBufLen]byte
	n, err := pin.direction.ReadAt0(buf[:])
	if err != nil {
		if err != io.EOF {
			return "", wrapPinError(pin, "get direction", err)
		}
	}
	return Direction(trimNewlines(buf[:n])), nil
}

// Edge is the signal edge that will make Interrupt send value to the channel.
type Edge string

const (
	// None means no edge is selected to generate interrupts.
	None Edge = "none"
	// Rising edges is is selected to generate interrupts. Rising: level is getting to high from low.
	Rising = "rising"
	// Falling edges is is selected to generate interrupts. Falling: level is getting to low from hight.
	Falling = "falling"
	// Both rising and falling edges are selected to generate interrupts.
	Both = "both"
)

// SetEdge sets which edges are selected to generate interrupts.
// Not all GPIO pins are configured to support edge selection,
// so, Edge should be called to confirm the desired edge are set actually.
func (pin *Pin) SetEdge(edge Edge) error {
	_, err := pin.edge.WriteAt0([]byte(edge))
	if err != nil {
		return wrapPinError(pin, "set edge", err)
	}
	return nil
}

// Edge returns which edges are selected to generate interrupts.
func (pin *Pin) Edge() (Edge, error) {
	var buf [strBufLen]byte
	n, err := pin.edge.ReadAt0(buf[:])
	if err != nil {
		if err != io.EOF {
			return "", wrapPinError(pin, "get edge", err)
		}
	}
	return Edge(trimNewlines(buf[:n])), nil
}

// Value returns the current value of the pin. 1 for high and 0 for low.
func (pin *Pin) Value() (byte, error) {
	var buf [1]byte
	n, err := pin.value.ReadAt0(buf[:])
	if err != nil {
		if !errors.Is(err, io.EOF) || n <= 0 {
			return 0, wrapPinError(pin, "get value", err)
		}
	}
	value := byte(0)
	if buf[0] != '0' {
		value = 1
	}
	return value, nil
}

// SetValue set the current value of the pin. 1 for high and 0 for low.
func (pin *Pin) SetValue(value byte) error {
	var buf = [1]byte{'1'}
	if value == 0 {
		buf[0] = '0'
	}
	if _, err := pin.value.WriteAt0(buf[:]); err != nil {
		return wrapPinError(pin, "set value", err)
	}
	return nil
}

// ActiveLow returns whether the pin is configured as active low.
func (pin *Pin) ActiveLow() (bool, error) {
	var buf [1]byte
	if _, err := pin.activeLow.ReadAt0(buf[:]); err != nil {
		return false, wrapPinError(pin, "get activelow", err)
	}
	return buf[0] == '1', nil
}

// SetActiveLow sets whether pin is configured as active low.
func (pin *Pin) SetActiveLow(value bool) error {
	var buf = [1]byte{'1'}
	if !value {
		buf[0] = '0'
	}
	if _, err := pin.activeLow.WriteAt0(buf[:]); err != nil {
		return wrapPinError(pin, "set activelow", err)
	}
	return nil
}

func writeExisting(path string, content string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write([]byte(content)); err != nil {
		return err
	}
	return nil
}

func trimNewlines(str []byte) string {
	return string(bytes.Trim(str, "\r\n"))
}

func wrapPinError(pin *Pin, action string, err error) error {
	return fmt.Errorf("failed to %v of GPIO pin #%v: %w, %s", action, pin.n, err, err)
}
