package app

import (
	"encoding/json"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-ignition/appliance/business/messages"
	"github.com/dumacp/go-ignition/appliance/business/services/grpc"
	svcpubsub "github.com/dumacp/go-ignition/appliance/business/services/pubsub"
	"github.com/dumacp/go-ignition/appliance/crosscutting/comm/pubsub"
	"github.com/dumacp/go-ignition/appliance/crosscutting/logs"
)

type App struct {
	path              string
	listenActor       *actor.PID
	gpsActor          *actor.PID
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
		return svcpubsub.NewService()
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
		_, err := ctx.SpawnNamed(propsListen, "listen-ignition")
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}

		propsGps := actor.PropsFromProducer(newGpsActor)
		app.gpsActor, err = ctx.SpawnNamed(propsGps, "gps-ignition")
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Println(err)
		}

		//if err := services(ctx); err != nil {
		//	time.Sleep(3 * time.Second)
		//	logs.LogError.Panic(err)
		//}

		logs.LogInfo.Printf("started %q actor", ctx.Self())

	case *actor.Stopping:
		logs.LogError.Printf("stopping actor, reason: %s", msg)
	case *messages.IgnitionEvent:
		//payload, err := msg.Marshal()
		if app.gpsActor != nil {
			req := ctx.RequestFuture(app.gpsActor, &MsgGPS{}, 1*time.Second)
			if err := req.Wait(); err != nil {
				logs.LogWarn.Print("empty gps frame in ignition event")
			}
			frame, err := req.Result()
			if err != nil {
				logs.LogWarn.Printf("empty gps frame in ignition event, error: %s", err)
			} else {
				if frame != nil {
					if data, ok := frame.([]byte); ok {
						msg.Value.Coord = string(data)
					}
				} else {
					logs.LogWarn.Print("empty gps frame in ignition event")
				}
			}
		}
		payload, err := json.Marshal(msg)
		if err != nil {
			logs.LogWarn.Printf("error publishing event: %s", err)
			break
		}
		pubsub.Publish(pubsub.TopicEvents, payload)
		logs.LogInfo.Printf("ignition event -> %s", payload)
		for _, subs := range app.eventSubscriptors {
			ctx.Send(subs, msg)
		}
	case *messages.IgnitionEventsSubscription:
		if ctx.Sender() != nil {
			app.eventSubscriptors[ctx.Sender().String()] = ctx.Sender()
		}
	}
}
