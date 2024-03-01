package app

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-ignition/internal/utils"
	"github.com/dumacp/go-ignition/pkg/messages"
	"github.com/dumacp/go-logs/pkg/logs"
)

type actorReboot struct {
	timeout    time.Duration
	lastDown   time.Time
	lastReboot time.Time
	cancel     func()
}

func NewRebootActor(time2reboot time.Duration) actor.Actor {

	a := &actorReboot{
		timeout:  time2reboot,
		lastDown: time.Now(),
	}

	return a
}

func (a *actorReboot) Receive(ctx actor.Context) {

	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("started %q actor", ctx.Self())
		if lastReboot, err := utils.Uptime(); err != nil {
			fmt.Printf("get last reboot error: %s\n", err)
		} else {
			fmt.Printf("get last reboot: %s\n", lastReboot)
			a.lastReboot = time.Now().Add(-lastReboot)
		}

	case *actor.Stopping:
		logs.LogError.Printf("stopping actor, reason: %s", msg)
		if a.cancel != nil {
			a.cancel()
		}

	case *messages.IgnitionEvent:
		if msg.GetValue() == nil {
			break
		}
		switch msg.GetValue().State {
		case messages.StateType_DOWN:
			a.lastDown = time.Now()
		case messages.StateType_UP:
			fmt.Printf("lastDown: %s, lastReboot: %s\n", a.lastDown, a.lastReboot)
			if time.Since(a.lastDown) > 20*time.Minute {
				if time.Since(a.lastReboot) > a.timeout {
					fmt.Println("reboot device")
				}
				go func() {
					time.Sleep(1 * time.Second)
					exec.Command("/sbin/reboot").CombinedOutput()
				}()
			}
		}
	}
}
