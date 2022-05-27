package messages

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-ignition/pkg/messages"
)

type MsgIngnitionRequest struct {
	Sender *actor.PID
}
type MsgIngnitionResponse struct {
	Msg *messages.IgnitionStateResponse
}
type MsgSubscriptionRequest struct {
	Sender *actor.PID
}
type MsgSubscriptionRemove struct {
	Sender *actor.PID
}
