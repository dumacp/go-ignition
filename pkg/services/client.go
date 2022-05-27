package services

import (
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/remote"
)

func NewService(r *remote.Remote, addr string) (*remote.ActorPidResponse, error) {
	return r.SpawnNamed(addr, fmt.Sprintf("client-%d", time.Now().Unix()), "client-ignition", 3*time.Second)
}
