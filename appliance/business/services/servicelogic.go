package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/eventstream"
	localmsg "github.com/dumacp/go-ignition/appliance/business/messages"
	"github.com/dumacp/go-ignition/appliance/services"
	"github.com/dumacp/go-ignition/appliance/services/messages"
)

type service struct {
	state messages.StatusResponse_StateType
}

var instance *service
var once sync.Once

//GetInstane get instance of service
func GetInstance() services.Service {
	if instance == nil {
		once.Do(func() {
			instance = &service{}
		})
	}
	return instance
}

func (svc *service) Start() {
	svc.state = messages.STARTED
	eventstream.Publish(&localmsg.Start{})
}

func (svc *service) Stop() {
	svc.state = messages.STOPPED
	eventstream.Publish(&localmsg.Stop{})
}

func (svc *service) Restart() {
	svc.state = messages.STOPPED
	eventstream.Publish(&localmsg.Stop{})
	time.Sleep(1 * time.Second)
	eventstream.Publish(&localmsg.Start{})
	svc.state = messages.STARTED
}

func (svc *service) Status() *messages.StatusResponse {
	return &messages.StatusResponse{
		State: svc.state,
	}
}

func (svc *service) Info(ctx actor.Context, pid *actor.PID) (*messages.IgnitionStateResponse, error) {
	future := ctx.RequestFuture(pid, &messages.IgnitionStateRequest{}, time.Second*3)
	err := future.Wait()
	if err != nil {
		return nil, err
	}
	res, err := future.Result()
	if err != nil {
		return nil, err
	}
	msg, ok := res.(*messages.IgnitionStateResponse)
	if !ok {
		return nil, fmt.Errorf("message error: %T", msg)
	}
	return msg, nil
}

func (svc *service) EventsSubscription(ctx actor.Context, pid *actor.PID) (*messages.IgnitionEventsSubscriptionAck, error) {
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
