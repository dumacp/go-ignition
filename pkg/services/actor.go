package services

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-ignition/pkg/messages"
)

type clientActor struct{}

func (a *clientActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *messages.IgnitionStateRequest:
		ctx.ActorSystem().EventStream.Publish(msg)
	}
}
