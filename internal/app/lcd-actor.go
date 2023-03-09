package app

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-ignition/pkg/messages"
	"github.com/dumacp/go-logs/pkg/logs"
)

type actorlcd struct {
	timeout time.Duration
	cancel  func()
}

func NewPowerActor(timeout time.Duration) actor.Actor {

	a := &actorlcd{
		timeout: timeout,
	}

	return a
}

func (a *actorlcd) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("started %q actor", ctx.Self())
	case *actor.Stopping:
		logs.LogError.Printf("stopping actor, reason: %s", msg)
		if a.cancel != nil {
			a.cancel()
		}
	case *messages.PowerEvent:
		ctx.Send(ctx.Parent(), msg)
	case *messages.IgnitionEvent:
		if msg.Value.State == messages.StateType_DOWN {
			if a.cancel != nil {
				a.cancel()
			}
			contx, cancel := context.WithCancel(context.TODO())
			a.cancel = cancel
			go down(contx, ctx, a.timeout)
		} else if msg.Value.State == messages.StateType_UP {
			logs.LogInfo.Println("power on devices with ignition signal")
			fmt.Println("power on devices with ignition signal")
			exec.Command("/bin/sh", "-c", "echo 0 > /sys/class/backlight/backlight-lvds/bl_power").Run()
			exec.Command("/bin/sh", "-c", "echo 1 > /sys/class/leds/enable-qr/brightness").Run()
			exec.Command("/bin/sh", "-c", "echo 1 > /sys/class/leds/enable-reader/brightness").Run()
			if a.cancel != nil {
				a.cancel()
			}
			ctx.Send(ctx.Self(), &messages.PowerEvent{
				Value:     messages.StateType_UP,
				Timestamp: time.Now().UnixMilli(),
			})
		}

	}
}

func down(contx context.Context, ctx actor.Context, timeout time.Duration) {

	self := ctx.Self()
	rootctx := ctx.ActorSystem().Root

	t1 := time.NewTimer(timeout)
	defer t1.Stop()
	select {
	case <-t1.C:
		logs.LogInfo.Println("power off devices with ignition signal")
		fmt.Println("power off devices with ignition signal")
		if err := exec.Command("/bin/sh", "-c", "echo 1 > /sys/class/backlight/backlight-lvds/bl_power").Run(); err != nil {
			logs.LogWarn.Printf("error with command lcd off: %s", err)
		}
		if err := exec.Command("/bin/sh", "-c", "echo 0 > /sys/class/leds/enable-qr/brightness").Run(); err != nil {
			logs.LogWarn.Printf("error with command qr off: %s", err)
		}
		if err := exec.Command("/bin/sh", "-c", "echo 0 > /sys/class/leds/enable-reader/brightness").Run(); err != nil {
			logs.LogWarn.Printf("error with command reader off: %s", err)
		}
		rootctx.Send(self, &messages.PowerEvent{
			Value:     messages.StateType_DOWN,
			Timestamp: time.Now().UnixMilli(),
		})
	case <-contx.Done():
	}
}
