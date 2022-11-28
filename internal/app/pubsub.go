package app

import (
	"encoding/json"

	"github.com/dumacp/go-ignition/pkg/messages"
)

const (
	SUBS_TOPIC = "SCHSERVICES/subscribe"
)

type ExternalSubscribe struct {
	ID     string
	Addres string
}

func Discover(msg []byte) interface{} {

	subs := new(messages.DiscoverIgnition)

	if err := json.Unmarshal(msg, subs); err != nil {
		return err
	}

	return subs
}
