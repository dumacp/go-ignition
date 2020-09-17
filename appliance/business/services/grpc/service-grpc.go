package grpc

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-ignition/appliance/business/messages"
	svc "github.com/dumacp/go-ignition/appliance/business/services"
	"github.com/dumacp/go-ignition/appliance/crosscutting/logs"
	"github.com/dumacp/go-ignition/appliance/services"
)

//Gateway interface
type Gateway interface {
	Receive(ctx actor.Context)
}

type grpcActor struct {
	svc services.Service
	// ctx actor.Context
}

//NewService create Service actor
func NewService() Gateway {
	act := &grpcActor{svc: svc.GetInstance()}
	//TODO:
	return act
}

//Receive function
func (act *grpcActor) Receive(ctx actor.Context) {
	// act.ctx = ctx
	switch ctx.Message().(type) {
	case *actor.Started:
		// receptionist.
	case *messages.Start:
		act.svc.Start()
	case *messages.Stop:
		act.svc.Stop()
	case *messages.Restart:
		act.svc.Restart()
	case *messages.StatusRequest:
		msg := act.svc.Status()
		ctx.Respond(msg)
	case *messages.IgnitionStateRequest:
		msg, err := act.svc.Info(ctx, ctx.Parent())
		if err != nil {
			logs.LogError.Println(err)
			break
		}
		ctx.Respond(msg)
	case *messages.IgnitionEventsSubscription:
		msg, err := act.svc.EventsSubscription(ctx, ctx.Parent())
		if err != nil {
			logs.LogWarn.Println(err)
			break
		}
		ctx.Respond(msg)
	}
}
