package app

import (
	"bytes"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-ignition/internal/pubsub"
	"github.com/dumacp/go-logs/pkg/logs"
)

type gpsActor struct {
	memGprmc *gpsFrame
	memGpgga *gpsFrame
}

func newGpsActor() actor.Actor {
	return &gpsActor{}
}

func (a *gpsActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("actor %q started", ctx.Self().GetId())
		pubsub.Subscribe("GPS", ctx.Self(), parseGps)
	case *actor.Stopped:
		logs.LogInfo.Printf("actor %q stopped", ctx.Self().GetId())
	case *gpsFrame:
		//logs.LogBuild.Printf("GPS frame -> %q", msg)
		switch {
		case bytes.HasPrefix(msg.data, []byte("$GPRMC")):
			a.memGprmc = msg
		case bytes.HasPrefix(msg.data, []byte("$GPGGA")):
			a.memGpgga = msg
		}
	case *MsgGPS:
		switch {
		case a.memGprmc != nil && a.memGprmc.timestamp.After(time.Now().Add(-30*time.Second)):
			ctx.Send(ctx.Sender(), a.memGprmc.data)
			logs.LogBuild.Printf("GPS send -> %q", a.memGprmc.data)
		case a.memGpgga != nil && a.memGpgga.timestamp.After(time.Now().Add(-30*time.Second)):
			ctx.Send(ctx.Sender(), a.memGpgga.data)
			logs.LogBuild.Printf("GPS send -> %q", a.memGpgga.data)
		default:
			ctx.Send(ctx.Sender(), nil)
		}
	}
}

type MsgGPS struct{}

type gpsFrame struct {
	data      []byte
	timestamp time.Time
}

func parseGps(msg []byte) interface{} {
	return &gpsFrame{data: msg, timestamp: time.Now()}
}
