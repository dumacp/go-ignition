package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/remote"
	"github.com/dumacp/go-ignition/appliance/business"
	"github.com/dumacp/go-ignition/appliance/crosscutting/logs"
)

const (
	port       = 8090
	pathEvents = "/dev/input/event0"
)

func main() {

	defer func() {
		if r := recover(); r != nil {
			logs.LogWarn.Printf("recover: %v", r)
		}
	}()

	portlocal := port
	for {
		portlocal++

		socket := fmt.Sprintf("127.0.0.1:%d", portlocal)
		testConn, err := net.DialTimeout("tcp", socket, 3*time.Second)
		if err != nil {
			break
		}
		logs.LogWarn.Printf("socket busy -> \"%s\"", socket)
		testConn.Close()
		time.Sleep(3 * time.Second)
	}

	remote.Start(fmt.Sprintf("127.0.0.1:%d", portlocal))

	rootContext := actor.EmptyRootContext

	propsApp := actor.PropsFromProducer(func() actor.Actor { return business.NewApp(pathEvents) })
	rootContext.SpawnNamed(propsApp, "ignition")

	finish := make(chan os.Signal, 1)
	signal.Notify(finish, syscall.SIGINT)
	signal.Notify(finish, syscall.SIGTERM)

	for {
		select {
		case <-finish:
			log.Print("Finish")
			return
		}
	}
}
