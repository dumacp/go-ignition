package app

import (
	"context"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-ignition/internal/device"
	"github.com/dumacp/go-ignition/internal/gpiosysfs"
	"github.com/dumacp/go-ignition/pkg/messages"
	"github.com/dumacp/go-logs/pkg/logs"
)

// ListenActor actor to listen events
type listenActorGpio struct {
	context     actor.Context
	path        int
	timeFailure int
	dev         *gpiosysfs.Pin
	cancel      func()
}

// NewListen create listen actor
func NewListenGpio(path int) actor.Actor {
	act := &listenActorGpio{}
	act.path = path
	act.timeFailure = 3
	return act
}

// Receive func Receive in actor
func (act *listenActorGpio) Receive(ctx actor.Context) {
	act.context = ctx
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("started \"%s\"", ctx.Self().GetId())
		dev, err := gpiosysfs.OpenPin(act.path)
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panicln(err)
		}
		logs.LogInfo.Printf("connected with serial port: %s", act.path)
		act.dev = dev
		contxt, cancel := context.WithCancel(context.Background())
		act.cancel = cancel
		go act.runListen(contxt)
	case *actor.Stopping:
		logs.LogWarn.Printf("\"%s\" - Stopped actor, reason -> %v", ctx.Self(), msg)
		if act.cancel != nil {
			act.cancel()
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

func (act *listenActorGpio) runListen(contxt context.Context) {

	if err := act.dev.SetDirection(gpiosysfs.In); err != nil {
		act.context.Send(act.context.Self(), &msgListenError{})
		return
	}
	time.Sleep(20 * time.Second)
	initialValue, err := act.dev.Value()
	if err != nil {
		fmt.Printf("initial value error: %s", err)
		logs.LogWarn.Printf("initial value error: %s", err)
		return
	} else {
		if initialValue != 0 {
			act.context.Send(act.context.Self(), &device.EventUP{})
		} else {
			act.context.Send(act.context.Self(), &device.EventDown{})
		}
	}

	events, err := act.dev.EpollEvents(contxt, true, gpiosysfs.Both)
	if err != nil {
		act.context.Send(act.context.Self(), &msgListenError{})
		return
	}
	for v := range events {
		fmt.Printf("listen event: %#v\n", v)
		if v.RisingEdge {
			act.context.Send(act.context.Self(), &device.EventUP{})
		} else {
			act.context.Send(act.context.Self(), &device.EventDown{})
		}
	}
	act.context.Send(act.context.Self(), &msgListenError{})
}
