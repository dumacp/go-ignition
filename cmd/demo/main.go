package main

import (
	"context"
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
	"github.com/dumacp/go-ignition/pkg/messages"
	"github.com/dumacp/go-logs/pkg/logs"
)

var timeout_lcd int

func init() {
	flag.IntVar(&timeout_lcd, "timeout-devices-off", 30, "timeout lcd power off (in seconds)")
}

func main() {

	defer func() {
		if r := recover(); r != nil {
			logs.LogWarn.Printf("recover: %v", r)
		}
	}()

	flag.Parse()

	portlocal := portlocal()

	sys := actor.NewActorSystem()
	config := remote.Configure("127.0.0.1", portlocal)

	r := remote.NewRemote(sys, config)
	r.Start()
	rootContext := sys.Root

	if err := pubsub.Init(rootContext); err != nil {
		log.Fatalln(err)
	}

	timeout := time.Duration(timeout_lcd) * time.Second
	propsApp := actor.PropsFromProducer(func() actor.Actor { return app.NewApp(&listenDemo{}, timeout, 30*time.Minute) })
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

	contxt, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := UpDownNoPrompt(contxt, "enviar evento, ", DOWN)

	for {
		select {
		case <-finish:
			cancel()
			log.Print("Finish")
			time.Sleep(300 * time.Millisecond)
			return
		case v := <-ch:
			var event *messages.IgnitionEvent
			if v == UP {
				event = &messages.IgnitionEvent{Value: &messages.ValueEvent{State: messages.StateType_UP, Coord: ""}, Timestamp: float64(float64(time.Now().UnixNano()) / 1000000000), Type: "IgnitionEvent"}
			} else {
				event = &messages.IgnitionEvent{Value: &messages.ValueEvent{State: messages.StateType_DOWN, Coord: ""}, Timestamp: float64(float64(time.Now().UnixNano()) / 1000000000), Type: "IgnitionEvent"}
			}
			fmt.Printf("ignition event -> %+v\n", event)
			rootContext.Send(pidApp, event)
		}
	}
}
