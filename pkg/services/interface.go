package services

import (
	"github.com/dumacp/go-ignition/pkg/messages"
)

//Service interface
type Service interface {
	//Start
	Start()
	Stop()
	Restart()
	Status() (*messages.StatusResponse, error)

	Info() (*messages.IgnitionStateResponse, error)
	EventsSubscription() error
}
