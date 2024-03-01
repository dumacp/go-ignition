package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-ignition/internal/pubsub"
	"github.com/dumacp/go-ignition/pkg/messages"
	"github.com/dumacp/go-logs/pkg/logs"
)

type App struct {
	timeout time.Duration

	propsGps          *actor.Props
	propsListen       *actor.Props
	propsPower        *actor.Props
	propsReboot       *actor.Props
	pidPower          *actor.PID
	gpsActor          *actor.PID
	pidReboot         *actor.PID
	eventSubscriptors map[string]*actor.PID
	powerSubscriptors map[string]*actor.PID
	lastEvent         *messages.IgnitionEvent
	lastPower         *messages.PowerEvent
}

// NewApp new actor
func NewApp(listen actor.Actor, timeout, time2reboot time.Duration) *App {
	app := &App{}
	app.timeout = timeout
	app.eventSubscriptors = make(map[string]*actor.PID)
	app.powerSubscriptors = make(map[string]*actor.PID)

	app.propsListen = actor.PropsFromFunc(listen.Receive)
	app.propsGps = actor.PropsFromProducer(newGpsActor)
	app.propsReboot = actor.PropsFromFunc(NewRebootActor(time2reboot).Receive)
	app.propsPower = actor.PropsFromProducer(func() actor.Actor {
		return NewPowerActor(app.timeout)
	})
	return app
}

// Receive function Receive
func (app *App) Receive(ctx actor.Context) {
	fmt.Printf("message arrive to %s. message type: %T, message body: %v, sender: %s\n",
		ctx.Self().GetId(), ctx.Message(), ctx.Message(), ctx.Sender().GetId())
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Println("starting actor")

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

		app.pidReboot, err = ctx.SpawnNamed(app.propsReboot, "reboot-ignition")
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Println(err)
		}

		logs.LogInfo.Printf("started %q actor", ctx.Self())

	case *actor.Stopping:
		logs.LogError.Printf("stopping actor, reason: %s", msg)
	case *messages.IgnitionStateRequest:
		if ctx.Sender() == nil && app.lastEvent == nil {
			break
		}

		ctx.Respond(&messages.IgnitionEvent{
			Value:     app.lastEvent.GetValue(),
			Timestamp: app.lastEvent.GetTimestamp(),
			Type:      app.lastEvent.GetType(),
		})
	case *messages.PowerStateRequest:
		if app.lastPower != nil && ctx.Sender() != nil {
			mss := &messages.PowerEvent{
				Value:     app.lastPower.GetValue(),
				Timestamp: app.lastPower.GetTimestamp(),
			}
			ctx.Respond(mss)
		}
	case *messages.DiscoverIgnition:
		if len(msg.GetAddr()) <= 0 || len(msg.GetId()) <= 0 {
			break
		}
		pid := actor.NewPID(msg.GetAddr(), msg.GetId())
		ctx.Request(pid, &messages.DiscoverResponseIgnition{
			Id:      ctx.Self().GetId(),
			Addr:    ctx.Self().GetAddress(),
			Timeout: int64(app.timeout.Milliseconds()),
		})
	case *messages.IgnitionEventsSubscription:
		if ctx.Sender() != nil {
			app.eventSubscriptors[ctx.Sender().String()] = ctx.Sender()
			if app.lastEvent != nil {
				mss := &messages.IgnitionEvent{
					Value:     app.lastEvent.GetValue(),
					Timestamp: app.lastEvent.GetTimestamp(),
					Type:      app.lastEvent.GetType(),
				}
				ctx.Send(ctx.Sender(), mss)
			}
		}
	case *messages.IgnitionPowerSubscription:
		if ctx.Sender() != nil {
			app.powerSubscriptors[ctx.Sender().String()] = ctx.Sender()
			if app.lastPower != nil {
				mss := &messages.PowerEvent{
					Value:     app.lastPower.GetValue(),
					Timestamp: app.lastPower.GetTimestamp(),
				}
				ctx.Send(ctx.Sender(), mss)
			}
		}
	case *messages.PowerEvent:
		app.lastPower = msg
		logs.LogInfo.Printf("power event -> %s", msg)
		fmt.Printf("power event -> %s\n", msg)
		for _, subs := range app.powerSubscriptors {
			mss := &messages.PowerEvent{
				Value:     msg.GetValue(),
				Timestamp: msg.GetTimestamp(),
			}
			ctx.Send(subs, mss)
		}
	case *messages.IgnitionEvent:

		app.lastEvent = msg

		if app.pidPower != nil {
			ctx.Send(app.pidPower, msg)
		}
		if app.pidReboot != nil {
			ctx.Send(app.pidReboot, msg)
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
		fmt.Printf("ignition event -> %s\n", payload)
		for _, subs := range app.eventSubscriptors {
			ctx.Send(subs, msg)
		}
	}
}
