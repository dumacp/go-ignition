package services

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	mss "github.com/dumacp/go-ignition/internal/messages"
	"github.com/dumacp/go-ignition/pkg/messages"
)

type clientActor struct{}

func NewClient() actor.Actor {
	return &clientActor{}
}

func (a *clientActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case *actor.Stopping:
		sendMsg := &mss.MsgSubscriptionRemove{
			Sender: ctx.Sender(),
		}
		ctx.ActorSystem().EventStream.Publish(sendMsg)
	case *messages.IgnitionStateRequest:
		sendMsg := &mss.MsgIngnitionRequest{
			Sender: ctx.Sender(),
		}
		ctx.ActorSystem().EventStream.Publish(sendMsg)
	case *messages.IgnitionEventsSubscription:
		sendMsg := &mss.MsgSubscriptionRequest{
			Sender: ctx.Sender(),
		}
		ctx.ActorSystem().EventStream.Publish(sendMsg)
	}
}
