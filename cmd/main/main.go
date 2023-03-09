package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/remote"
	"github.com/dumacp/go-ignition/internal/app"
	"github.com/dumacp/go-ignition/internal/pubsub"
	"github.com/dumacp/go-ignition/pkg/ignition"
	"github.com/dumacp/go-logs/pkg/logs"
)

const (
	port        = 8090
	pathEvents  = "/dev/input/event0"
	showVersion = "1.1.1_test"
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

	portlocal := portlocal()

	sys := actor.NewActorSystem()
	config := remote.Configure("127.0.0.1", portlocal)

	r := remote.NewRemote(sys, config)
	r.Start()
	rootContext := sys.Root

	if err := pubsub.Init(rootContext); err != nil {
		log.Fatalln(err)
	}

	timeout := time.Duration(timeout_lcd) * time.Minute
	propsApp := actor.PropsFromProducer(func() actor.Actor { return app.NewApp(pathEvents, timeout) })
	pidApp, err := rootContext.SpawnNamed(propsApp, "ignition")
	if err != nil {
		log.Fatalln(err)
	}

	if err := pubsub.Subscribe(ignition.DISCV_TOPIC, pidApp, app.Discover); err != nil {
		logs.LogError.Fatalln(err)
	}

	finish := make(chan os.Signal, 1)
	signal.Notify(finish, syscall.SIGINT)
	signal.Notify(finish, syscall.SIGTERM)

	for range finish {
		log.Print("Finish")
		time.Sleep(300 * time.Millisecond)
		return
	}
}
