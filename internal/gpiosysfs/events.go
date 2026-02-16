package gpiosysfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

const (
	EPOLLET        = 1 << 31
	MaxEpollEvents = 32
)

// FdEvents converts epoll_wait loops to a chanel.
type FdEvents struct {
	events chan *Event
	err    error
	cancel func()
}

func (ev *FdEvents) Events() <-chan *Event {
	return ev.events
}

// Event is a GPIO event.
type Event struct {
	Time       time.Time // The best estimate of time of event occurrence.
	Pin        int32
	RisingEdge bool // Whether this event is triggered by a rising edge.
}

func newEvents(ctx context.Context, f *gpioFile, pin int32) (*FdEvents, error) {

	fmt.Println("listen events")
	var err error
	if f.rfile == nil {
		f.rfile, err = os.Open(f.Path)
		if err != nil {
			f.rfile = nil
			return nil, err
		}
	}

	fdSet := &unix.FdSet{}
	fdSet.Bits[int(f.rfile.Fd()/64)] |= 1 << uint(f.rfile.Fd()%64)

	// bucle infinito para esperar cambios en el estado de la GPIO
	ctx, cancel := context.WithCancel(ctx)
	events := new(FdEvents)
	events.events = make(chan *Event, 1)
	go func() {
		defer cancel()

		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		value := -1

		for {
			select {
			case <-ctx.Done():
				close(events.events)
			case <-tick.C:
				// configurar el conjunto de descriptores de archivo para esperar en
				timeout := unix.NsecToTimeval(10000000) // establecer un tiempo de espera de 10 ms
				ready, err := unix.Select(int(f.rfile.Fd()+1), fdSet, nil, nil, &timeout)
				if err != nil {
					events.err = err
					return
				}

				// verificar si la llamada a select() ha expirado o si hay descriptores de archivo listos
				if ready > 0 {
					// leer el valor del archivo para determinar el nuevo estado de la GPIO
					// var buf [1]byte
					// _, err = gpioFile.Read(buf[:])
					// if err != nil {
					//     panic(err)
					// }

					// leer el valor del archivo para determinar el nuevo estado de la GPIO
					buf := make([]byte, 1)
					n, err := f.rfile.ReadAt(buf[:], 0)
					if err != nil {
						if !errors.Is(err, io.EOF) || n <= 0 {
							events.err = err
							return
						}
					}
					// fmt.Printf("gpio( %q ) = %d\n", f.Path, int(buf[0]))

					// imprimir el valor actual de la GPIO
					if value != int(buf[0]) {
						value = int(buf[0])
						select {
						case <-events.events:
						default:
						}
						events.events <- &Event{
							Pin:        pin,
							RisingEdge: value == 0x30,
							Time:       time.Now(),
						}
					}
				}

			}
		}
	}()

	return events, nil

}

// PinWithEvent is an opened GPIO pin whose events can be read.
type PinWithEvent struct {
	*Pin
	events *FdEvents
}

// OpenPinWithEvents opens a GPIO pin for input and GPIO events.
func OpenPinWithEvents(ctx context.Context, n int) (*PinWithEvent, error) {
	p, err := OpenPin(n)
	if err != nil {
		return nil, err
	}
	err = p.SetDirection(In)
	if err != nil {
		return nil, err
	}

	events, err := newEvents(ctx, p.value, int32(n))
	if err != nil {
		return nil, err
	}

	pin := &PinWithEvent{
		Pin:    p,
		events: events,
	}

	return pin, nil
}

func (pin *PinWithEvent) Close() (err error) {
	// Close pin.events first.
	// pin.Pin is still used by pin.events before pin.events is closed.
	err1 := pin.events.Close()
	err2 := pin.Pin.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

// Close stops the epoll_wait loop, close the fd, and close the event channel.
func (events *FdEvents) Close() (err error) {
	if events.cancel != nil {
		events.cancel()
	}
	return nil
}

// Events returns a channel from which the occurrence time of GPIO events can be read.
// The GPIO events of this pin will be sent to the returned channel, and the channel is closed when l is closed.
//
// Package gpiosysfs will not block sending to the channel: it only keeps the lastest
// value in the channel.
func (pin *PinWithEvent) Events() <-chan *Event {
	return pin.events.Events()
}
