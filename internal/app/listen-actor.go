package app

import (
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-ignition/internal/device"
	"github.com/dumacp/go-ignition/pkg/messages"
	"github.com/dumacp/go-logs/pkg/logs"
	evdev "github.com/gvalkov/golang-evdev"
)

const (
	ignitionType = "IgnitionEvent"
)

// ListenActor actor to listen events
type listenActor struct {
	context actor.Context

	quit chan int

	path        string
	dev         *evdev.InputDevice
	timeFailure int
}

// NewListen create listen actor
func NewListen(path string) actor.Actor {
	act := &listenActor{}
	act.path = path
	act.quit = make(chan int, 0)
	act.timeFailure = 3
	return act
}

// Receive func Receive in actor
func (act *listenActor) Receive(ctx actor.Context) {
	act.context = ctx
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("started \"%s\"", ctx.Self().GetId())
		dev, err := device.NewEventDevice(act.path)
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panicln(err)
		}
		logs.LogInfo.Printf("connected with serial port: %s", act.path)
		act.dev = dev
		go act.runListen(act.quit)
	case *actor.Stopping:
		logs.LogWarn.Printf("\"%s\" - Stopped actor, reason -> %v", ctx.Self(), msg)
		select {
		case act.quit <- 1:
		case <-time.After(3 * time.Second):
		}
	case *actor.Restarting:
		logs.LogWarn.Printf("\"%s\" - Restarting actor, reason -> %v", ctx.Self(), msg)
	case *msgListenError:
		time.Sleep(time.Duration(act.timeFailure) * time.Second)
		act.timeFailure = 2 * act.timeFailure
		logs.LogError.Panicln("listen error")
	case *device.EventUP:
		event := &messages.IgnitionEvent{Value: &messages.ValueEvent{State: messages.StateType_UP, Coord: ""}, Timestamp: float64(float64(time.Now().UnixNano()) / 1000000000), Type: ignitionType}
		fmt.Printf("ignition event -> %+v\n", event)
		ctx.Send(ctx.Parent(), event)
	case *device.EventDown:
		event := &messages.IgnitionEvent{Value: &messages.ValueEvent{State: messages.StateType_DOWN, Coord: ""}, Timestamp: float64(float64(time.Now().UnixNano()) / 1000000000), Type: ignitionType}
		fmt.Printf("ignition event -> %+v\n", event)
		ctx.Send(ctx.Parent(), event)
	}
}

type msgListenError struct{}

func (act *listenActor) runListen(quit chan int) {
	events := device.Listen(quit, act.dev)
	for v := range events {
		fmt.Printf("listen event: %#v\n", v)
		switch event := v.(type) {
		case *device.EventUP:
			act.context.Send(act.context.Self(), event)
		case *device.EventDown:
			act.context.Send(act.context.Self(), event)
		}
	}
	act.context.Send(act.context.Self(), &msgListenError{})
}
