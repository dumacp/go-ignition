package app

import (
	"encoding/json"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/eventstream"
	mss "github.com/dumacp/go-ignition/internal/messages"
	"github.com/dumacp/go-ignition/internal/pubsub"
	"github.com/dumacp/go-ignition/pkg/messages"
	"github.com/dumacp/go-logs/pkg/logs"
)

type App struct {
	path    string
	timeout time.Duration
	// listenActor       *actor.PID
	gpsActor          *actor.PID
	propsGps          *actor.Props
	propsListen       *actor.Props
	propsPower        *actor.Props
	pidPower          *actor.PID
	eventSubscriptors map[string]*actor.PID
	lastEvent         *messages.IgnitionEvent
}

//NewApp new actor
func NewApp(path string, timeout time.Duration) *App {
	app := &App{path: path}
	app.timeout = timeout
	app.eventSubscriptors = make(map[string]*actor.PID)

	app.propsListen = actor.PropsFromProducer(func() actor.Actor {
		return NewListen(app.path)
	})
	app.propsGps = actor.PropsFromProducer(newGpsActor)
	app.propsPower = actor.PropsFromProducer(func() actor.Actor {
		return NewPowerActor(app.timeout)
	})
	return app
}

func subscribe(ctx actor.Context, evs *eventstream.EventStream) {
	rootctx := ctx.ActorSystem().Root
	// pid := ctx.Sender()
	self := ctx.Self()

	fn := func(evt interface{}) {
		rootctx.Send(self, evt)
	}
	sub := evs.Subscribe(fn)
	sub.WithPredicate(func(evt interface{}) bool {
		switch evt.(type) {
		case *mss.MsgIngnitionRequest:
			return true
		case *mss.MsgSubscriptionRemove:
			return true
		case *mss.MsgSubscriptionRequest:
			return true
		}
		return false
	})
}

// func services(ctx actor.Context) error {
// 	var err error
// 	propsGrpc := actor.PropsFromProducer(func() actor.Actor {
// 		return grpc.NewService(ctx.ActorSystem().Root)
// 	})
// 	_, err = ctx.SpawnNamed(propsGrpc, "svc-grpc")
// 	if err != nil {
// 		return err
// 	}
// 	propsPubSub := actor.PropsFromProducer(func() actor.Actor {
// 		return svcpubsub.NewService(ctx.ActorSystem().Root)
// 	})
// 	_, err = ctx.SpawnNamed(propsPubSub, "svc-mqtt")
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

//Receive function Receive
func (app *App) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Println("starting actor")
		subscribe(ctx, ctx.ActorSystem().EventStream)

		_, err := ctx.SpawnNamed(app.propsListen, "listen-ignition")
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}

		app.gpsActor, err = ctx.SpawnNamed(app.propsGps, "gps-ignition")
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Println(err)
		}

		app.pidPower, err = ctx.SpawnNamed(app.propsPower, "power-ignition")
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Println(err)
		}

		logs.LogInfo.Printf("started %q actor", ctx.Self())

	case *actor.Stopping:
		logs.LogError.Printf("stopping actor, reason: %s", msg)
	case *messages.IgnitionEvent:

		app.lastEvent = msg

		if app.pidPower != nil {
			ctx.Send(app.pidPower, msg)
		}

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
	case *mss.MsgSubscriptionRequest:
		if msg.Sender != nil {
			app.eventSubscriptors[msg.Sender.String()] = msg.Sender
		}
	case *mss.MsgIngnitionRequest:
		if msg.Sender != nil {
			if app.lastEvent != nil {
				ctx.Send(msg.Sender, app.lastEvent)
			}
		}
	case *mss.MsgSubscriptionRemove:
		if msg.Sender != nil {
			delete(app.eventSubscriptors, msg.Sender.String())
		}
	}
}
