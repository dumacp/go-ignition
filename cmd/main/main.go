package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/remote"
	"github.com/dumacp/go-ignition/internal/app"
	"github.com/dumacp/go-ignition/internal/pubsub"
	"github.com/dumacp/go-ignition/internal/services"
	"github.com/dumacp/go-logs/pkg/logs"
)

const (
	port        = 8090
	pathEvents  = "/dev/input/event0"
	showVersion = "1.0.4"
)

var debug bool
var logStd bool
var version bool
var timeout_lcd int

func init() {
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.BoolVar(&logStd, "logStd", false, "log in stderr")
	flag.IntVar(&timeout_lcd, "timeout-devices-off", 30, "timeout lcd power off (in minutes)")
	flag.BoolVar(&version, "version", false, "show version")
}

func main() {

	defer func() {
		if r := recover(); r != nil {
			logs.LogWarn.Printf("recover: %v", r)
		}
	}()

	flag.Parse()

	initLogs(debug, logStd)
	if version {
		fmt.Printf("version: %s\n", showVersion)
		os.Exit(-2)
	}
	logs.LogInfo.Printf("version: %s\n", showVersion)

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

	sys := actor.NewActorSystem()
	config := remote.Configure("127.0.0.1", portlocal)

	r := remote.NewRemote(sys, config)

	// r.Register("client-ignition", services.NewService)

	rootContext := sys.Root

	client := services.NewClient()

	propsClient := actor.PropsFromFunc(client.Receive)

	r.Register("client-ignition", propsClient)

	if err := pubsub.Init(rootContext); err != nil {
		log.Fatalln(err)
	}

	timeout := time.Duration(timeout_lcd) * time.Minute
	propsApp := actor.PropsFromProducer(func() actor.Actor { return app.NewApp(pathEvents, timeout) })
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
