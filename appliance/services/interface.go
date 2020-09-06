package services

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-ignition/appliance/services/messages"
)

//Service interface
type Service interface {
	//Start
	Start()
	Stop()
	Restart()
	Status() *messages.StatusResponse

	Info(ctx actor.Context, pid *actor.PID) (*messages.InfoCounterResponse, error)
}
