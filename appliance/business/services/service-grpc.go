package services

import (
	"log"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-ignition/appliance/services"
	"github.com/dumacp/go-ignition/appliance/services/messages"
)

//Gateway interface
type Gateway interface {
	Receive(ctx actor.Context)
}

type grpcActor struct {
	svc         services.Service
	log         *log.Logger
	ctx         actor.Context
	countingPID *actor.PID
	behavior    actor.Behavior
}

//NewService create Service actor
func NewService(svc services.Service) Gateway {
	act := &grpcActor{svc: svc}
	act.behavior.Become(act.Started)
	return act
}

//Receive function
func (act *grpcActor) Receive(ctx actor.Context) {
	act.ctx = ctx
	act.behavior.Receive(ctx)
}

func (act *grpcActor) Started(ctx actor.Context) {

	switch ctx.Message().(type) {
	case *messages.Start:
		act.svc.Start()
	case *messages.Stop:
		act.svc.Stop()
		act.behavior.Become(act.Stopped)
	case *messages.Restart:
		act.svc.Restart()
	case *messages.StatusRequest:
		msg := act.svc.Status()
		ctx.Respond(msg)
	case *messages.AddressCounterRequest:
		ctx.Send(
			ctx.Sender(),
			messages.AddressCounterResponse{
				ID:   ctx.Self().Id,
				Addr: ctx.Self().Address,
			})
	case *messages.InfoCounterRequest:
		msg, err := act.svc.Info(ctx, act.countingPID)
		if err != nil {
			act.log.Println(err)
			break
		}
		ctx.Respond(msg)
	}
}

func (act *grpcActor) Stopped(ctx actor.Context) {
	switch ctx.Message().(type) {
	case *messages.Start:
		act.svc.Start()
		act.behavior.Become(act.Started)
	case *messages.Restart:
		act.svc.Restart()
	}
}
