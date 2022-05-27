package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/dumacp/go-ignition/pkg/messages"
)

type Service struct {
	// rootctx *actor.RootContext
	state  messages.StatusResponse_State
	events *eventstream.EventStream
}

var instance *Service
var once sync.Once

//GetInstane get instance of service
func GetInstance(ctx actor.Context) *Service {
	if instance == nil {
		once.Do(func() {
			instance = &Service{}
			evs := ctx.ActorSystem().EventStream
			subs := evs.Subscribe(func(evt interface{}) {

			})
			subs.WithPredicate(func(evt interface{}) bool {
				switch evt.(type) {
				case *messages.IgnitionStateResponse:
					return true
				}
				return false
			})
			instance.events = evs
		})
	}
	return nil
}

func (svc *Service) Start() {
	svc.state = messages.STARTED
	svc.events.Publish(&messages.Start{})
}

func (svc *Service) Stop() {
	svc.state = messages.STOPPED
	svc.events.Publish(&messages.Stop{})
}

func (svc *Service) Restart() {
	svc.state = messages.STOPPED
	svc.events.Publish(&messages.Stop{})
	time.Sleep(1 * time.Second)
	svc.events.Publish(&messages.Start{})
	svc.state = messages.STARTED
}

func (svc *Service) Status() *messages.StatusResponse {
	return &messages.StatusResponse{
		Status: svc.state,
	}
}

// func (svc *Service) Info() (*messages.IgnitionStateResponse, error) {
// 	func(ctx actor.Context) {
// 		switch ctx.Message().(type) {
// 		case *messages.IgnitionStateResponse:

// 		}
// 	}
// 	svc.rootctx.RequestFuture()
// 	svc.events.Publish(&messages.IgnitionStateRequest{Sender: nil})

// 	return msg, nil
// }

func (svc *Service) EventsSubscription(ctx actor.Context, pid *actor.PID) (*messages.IgnitionEventsSubscriptionAck, error) {
	future := ctx.RequestFuture(pid, &messages.IgnitionEventsSubscription{}, time.Second*3)
	err := future.Wait()
	if err != nil {
		return nil, err
	}
	res, err := future.Result()
	if err != nil {
		return nil, err
	}
	msg, ok := res.(*messages.IgnitionEventsSubscriptionAck)
	if !ok {
		return nil, fmt.Errorf("message error: %T", msg)
	}
	return msg, nil
}
