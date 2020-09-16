package business

import (
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-ignition/appliance/business/device"
	"github.com/dumacp/go-ignition/appliance/business/services/grpc"
	"github.com/dumacp/go-ignition/appliance/business/services/pubsub"
	gwpubsub "github.com/dumacp/go-ignition/appliance/crosscutting/comm/pubsub"
	"github.com/dumacp/go-ignition/appliance/crosscutting/logs"
	evemsg "github.com/dumacp/go-ignition/appliance/events/messages"
	svcmsg "github.com/dumacp/go-ignition/appliance/services/messages"
)

type App struct {
	path              string
	listenActor       *actor.PID
	eventSubscriptors map[string]*actor.PID
}

//NewApp new actor
func NewApp(path string) *App {
	app := &App{path: path}
	app.eventSubscriptors = make(map[string]*actor.PID)
	return app
}

func services(ctx actor.Context) error {
	var err error
	propsGrpc := actor.PropsFromProducer(func() actor.Actor {
		return grpc.NewService()
	})
	_, err = ctx.SpawnNamed(propsGrpc, "svc-grpc")
	if err != nil {
		return err
	}
	propsPubSub := actor.PropsFromProducer(func() actor.Actor {
		return pubsub.NewService()
	})
	_, err = ctx.SpawnNamed(propsPubSub, "svc-mqtt")
	if err != nil {
		return err
	}
	return nil
}

//Receive function Receive
func (app *App) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Println("starting actor")
		propsListen := actor.PropsFromProducer(func() actor.Actor {
			return NewListen(app.path)
		})
		_, err := ctx.SpawnNamed(propsListen, "listen-actor")
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}

		if err := services(ctx); err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}

		logs.LogInfo.Println("started actor")

	case *actor.Stopping:
		logs.LogError.Printf("stopping actor, reason: %s", msg)
	case *device.EventUP:
		event := &evemsg.IgnitionUP{}
		payload, err := event.Marshal()
		if err != nil {
			logs.LogError.Println(err)
		}
		gwpubsub.Publish(gwpubsub.TopicEvents, payload)
		for _, subs := range app.eventSubscriptors {
			ctx.Send(subs, event)
		}
	case *svcmsg.IgnitionEventsSubscription:
		if ctx.Sender() != nil {
			app.eventSubscriptors[ctx.Sender().String()] = ctx.Sender()
		}
	}
}
