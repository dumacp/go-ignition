package app

import (
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-ignition/appliance/business/device"
	"github.com/dumacp/go-ignition/appliance/crosscutting/logs"
	"github.com/dumacp/go-ignition/appliance/events/messages"
	evdev "github.com/gvalkov/golang-evdev"
)

//ListenActor actor to listen events
type ListenActor struct {
	context actor.Context

	quit chan int

	path        string
	dev         *evdev.InputDevice
	timeFailure int
}

//NewListen create listen actor
func NewListen(path string) *ListenActor {
	act := &ListenActor{}
	act.path = path
	act.quit = make(chan int, 0)
	act.timeFailure = 3
	return act
}

//Receive func Receive in actor
func (act *ListenActor) Receive(ctx actor.Context) {
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
		event := messages.IgnitionEvent{Event: messages.UP, TimeStamp: time.Now().Unix()}
		//payload, err := event.Marshal()
		//if err != nil {
		//	logs.LogWarn.Printf("error publishing event: %s", err)
		//}
		//pubsub.Publish(pubsub.TopicEvents, payload)
		ctx.Send(ctx.Parent(), event)
	case *device.EventDown:
		event := messages.IgnitionEvent{Event: messages.DOWN, TimeStamp: time.Now().Unix()}
		ctx.Send(ctx.Parent(), event)
	}
}

type msgListenError struct{}

func (act *ListenActor) runListen(quit chan int) {
	events := device.Listen(quit, act.dev)
	for v := range events {
		logs.LogBuild.Printf("listen event: %#v\n", v)
		switch event := v.(type) {
		case *device.EventUP:
			act.context.Send(act.context.Self(), event)
		case *device.EventDown:
			act.context.Send(act.context.Self(), event)
		}
	}
	act.context.Send(act.context.Self(), &msgListenError{})
}
