package app

import (
	"os/exec"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-ignition/pkg/messages"
	"github.com/dumacp/go-logs/pkg/logs"
)

type actorlcd struct {
	timeout time.Duration
	quit    chan int
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
		if a.quit != nil {
			select {
			case _, ok := <-a.quit:
				if ok {
					close(a.quit)
				}
			default:
				close(a.quit)
			}
		}
		a.quit = make(chan int)

	case *actor.Stopping:
		logs.LogError.Printf("stopping actor, reason: %s", msg)
		close(a.quit)
	case *messages.IgnitionEvent:
		if a.quit != nil {
			select {
			case _, ok := <-a.quit:
				if ok {
					close(a.quit)
				}
			default:
				close(a.quit)
			}
			time.Sleep(10 * time.Millisecond)
		}
		a.quit = make(chan int)
		if msg.Value.State == messages.DOWN {
			go down(a.quit, a.timeout)
		} else if msg.Value.State == messages.UP {
			logs.LogInfo.Println("power on devices with ignition signal")
			exec.Command("/bin/sh", "-c", "echo 0 > /sys/class/backlight/backlight-lvds/bl_power").Run()
			exec.Command("/bin/sh", "-c", "echo 1 > /sys/class/leds/enable-qr/brightness").Run()
			close(a.quit)
		}
	case *messages.IgnitionEventsSubscription:

	}
}

func down(quit chan int, timeout time.Duration) {

	t1 := time.NewTimer(timeout)
	defer t1.Stop()
	select {
	case <-t1.C:
		logs.LogInfo.Println("power off devices with ignition signal")
		exec.Command("/bin/sh", "-c", "echo 1 > /sys/class/backlight/backlight-lvds/bl_power").Run()
		exec.Command("/bin/sh", "-c", "echo 0 > /sys/class/leds/enable-qr/brightness").Run()
	case <-quit:
	}
}
