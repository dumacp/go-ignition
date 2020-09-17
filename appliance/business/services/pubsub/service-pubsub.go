package pubsub

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-ignition/appliance/business/messages"
	svc "github.com/dumacp/go-ignition/appliance/business/services"
	"github.com/dumacp/go-ignition/appliance/crosscutting/comm/pubsub"
	"github.com/dumacp/go-ignition/appliance/crosscutting/logs"
	"github.com/dumacp/go-ignition/appliance/services"
)

//Gateway interface
type Gateway interface {
	Receive(ctx actor.Context)
}

type pubsubActor struct {
	svc services.Service
	ctx actor.Context
}

//NewService create Service actor
func NewService() Gateway {
	act := &pubsubActor{svc: svc.GetInstance()}
	return act
}

func service(ctx actor.Context) {
	pubsub.Subscribe(pubsub.TopicStart, func(msg []byte) {
		ctx.Send(ctx.Self(), &messages.Start{})
	})
	pubsub.Subscribe(pubsub.TopicStop, func(msg []byte) {
		ctx.Send(ctx.Self(), &messages.Stop{})
	})
	pubsub.Subscribe(pubsub.TopicRestart, func(msg []byte) {
		ctx.Send(ctx.Self(), &messages.Restart{})
	})
	pubsub.Subscribe(pubsub.TopicStatus, func(msg []byte) {
		req := &messages.StatusRequest{}
		if err := req.Unmarshal(msg); err != nil {
			logs.LogWarn.Println(err)
			return
		}
		ctx.Send(ctx.Self(), req)
	})
	pubsub.Subscribe(pubsub.TopicRequestInfoState, func(msg []byte) {
		req := &messages.InfoCounterRequest{}
		if err := req.Unmarshal(msg); err != nil {
			logs.LogWarn.Println(err)
			return
		}
		ctx.Send(ctx.Self(), req)
	})

}

//Receive function
func (act *pubsubActor) Receive(ctx actor.Context) {
	act.ctx = ctx
	switch msg := ctx.Message().(type) {
	case *messages.Start:
		act.svc.Start()
	case *messages.Stop:
		act.svc.Stop()
	case *messages.Restart:
		act.svc.Restart()
	case *messages.StatusRequest:
		resp := act.svc.Status()
		payload, err := resp.Marshal()
		if err != nil {
			logs.LogWarn.Println(err)
			break
		}
		pubsub.Publish(fmt.Sprintf("%s/%s", pubsub.TopicStatus, msg.GetSender()), payload)
	case *messages.IgnitionStateRequest:
		resp, err := act.svc.Info(ctx, ctx.Parent())
		if err != nil {
			logs.LogError.Println(err)
			break
		}
		payload, err := resp.Marshal()
		if err != nil {
			logs.LogWarn.Println(err)
			break
		}
		pubsub.Publish(fmt.Sprintf("%s/%s", pubsub.TopicRequestInfoState, msg.GetSender()), payload)
	}
}
