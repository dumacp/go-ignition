package gpiosysfs

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"
)

func (p *Pin) EpollEvents(ctx context.Context, activeLow bool, edge Edge) (chan Event, error) {

	fmt.Println("listen events")

	if err := p.SetDirection(In); err != nil {
		return nil, err
	}
	if err := p.SetActiveLow(activeLow); err != nil {
		return nil, err
	}
	if err := p.SetEdge(edge); err != nil {
		return nil, err
	}

	return newEpollEvents(ctx, p)

}

func newEpollEvents(ctx context.Context, pin *Pin) (chan Event, error) {

	if pin.value.rfile == nil {
		var err error
		pin.value.rfile, err = os.Open(pin.value.Path)
		if err != nil {
			pin.value.rfile = nil
			return nil, err
		}
	}

	f := pin.value.rfile
	fd := f.Fd()

	// Crea una instancia epoll
	epollFd, err := syscall.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	// Configura el interés en el descriptor de archivo para eventos EPOLLPRI (eventos urgentes)
	event := syscall.EpollEvent{
		Fd:     int32(fd),
		Events: syscall.EPOLLPRI | syscall.EPOLLERR,
	}
	if err = syscall.EpollCtl(epollFd, syscall.EPOLL_CTL_ADD, int(fd), &event); err != nil {
		return nil, err
	}

	events := make([]syscall.EpollEvent, 1)
	timeout := 10000 // 10 milisegundos

	ch := make(chan Event, 1)
	go func() {
		defer close(ch)
		defer syscall.Close(epollFd)
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Context finalizado")
				return
			default:
				n, err := syscall.EpollWait(epollFd, events, timeout)
				if err != nil {
					fmt.Printf("error: %s\n", err)
					continue
					// return
				}

				for i := 0; i < n; i++ {
					if events[i].Events&syscall.EPOLLPRI != 0 {
						// Ha ocurrido un cambio en el archivo. Procesa el cambio aquí.
						fmt.Println("Cambio detectado en", pin.value.Path)
						bff := make([]byte, 1)
						if _, err := f.ReadAt(bff, 0); err != nil {
							fmt.Println("error leyendo GPIO")
							return
						}
						select {
						case ch <- Event{
							Time:       time.Now(),
							Pin:        int32(pin.n),
							RisingEdge: bff[0] == 0x30,
						}:
						default:
							fmt.Println("timeout leyendo GPIO")
						}

					}
				}
			}
		}
	}()

	return ch, nil

}
