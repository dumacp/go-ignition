package business

import (
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-ignition/appliance/business/device"
	"github.com/dumacp/go-ignition/appliance/business/messages"
	"github.com/dumacp/go-ignition/appliance/crosscutting/logs"
	evdev "github.com/gvalkov/golang-evdev"
)

//ListenActor actor to listen events
type ListenActor struct {
	context       actor.Context
	enters0Before int64
	exits0Before  int64
	locks0Before  int64
	enters1Before int64
	exits1Before  int64
	locks1Before  int64

	quit chan int

	socket      string
	baudRate    int
	dev         *evdev.InputDevice
	timeFailure int
}

//NewListen create listen actor
func NewListen(socket string, baudRate int, countingActor *actor.PID) *ListenActor {
	act := &ListenActor{}
	act.socket = socket
	act.baudRate = baudRate
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
		dev, err := device.NewEventDevice(act.socket)
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panicln(err)
		}
		logs.LogInfo.Printf("connected with serial port: %s", act.socket)
		act.dev = dev
		go act.runListen(act.quit)
	case *actor.Stopping:
		logs.LogWarn.Println("stopped actor")
		select {
		case act.quit <- 1:
		case <-time.After(3 * time.Second):
		}
	case *msgListenError:
		time.Sleep(time.Duration(act.timeFailure) * time.Second)
		act.timeFailure = 2 * act.timeFailure
		logs.LogError.Panicln("listen error")
	}
}

type msgListenError struct{}

func (act *ListenActor) runListen(quit chan int) {
	events := device.Listen(quit, act.dev)
	for v := range events {
		logs.LogBuild.Printf("listen event: %#v\n", v)
		switch event := v.(type) {
		case *device.EventUP:
			if event.id == 0 {
				enters := event.value
				if diff := enters - act.enters0Before; diff > 0 {
					act.context.Send(act.countingActor, &messages.Event{Id: 0, Type: messages.INPUT, Value: enters})
				}
				act.enters0Before = enters
			} else {
				enters := event.value
				if diff := enters - act.enters1Before; diff > 0 {
					act.context.Send(act.countingActor, &messages.Event{Id: 1, Type: messages.INPUT, Value: enters})
				}
				act.enters1Before = enters
			}
		case *device.EventDown:
			if event.id == 0 {
				enters := event.value
				if diff := enters - act.exits0Before; diff > 0 {
					act.context.Send(act.countingActor, &messages.Event{Id: 0, Type: messages.OUTPUT, Value: enters})
				}
				act.exits0Before = enters
			} else {
				enters := event.value
				if diff := enters - act.exits1Before; diff > 0 {
					act.context.Send(act.countingActor, &messages.Event{Id: 1, Type: messages.OUTPUT, Value: enters})
				}
				act.exits1Before = enters
			}
		case *Lock:
			if event.id == 0 {
				enters := event.value
				if diff := enters - act.locks0Before; diff > 0 {
					act.context.Send(act.countingActor, &messages.Event{Id: 0, Type: messages.TAMPERING, Value: enters})
				}
				act.locks0Before = enters
			} else {
				enters := event.value
				if diff := enters - act.locks1Before; diff > 0 {
					act.context.Send(act.countingActor, &messages.Event{Id: 1, Type: messages.TAMPERING, Value: enters})
				}
				act.locks1Before = enters
			}
		}
	}
	act.context.Send(act.context.Self(), &msgListenError{})
}
