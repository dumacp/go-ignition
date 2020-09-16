package pubsub

import (
	"fmt"
	"sync"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-ignition/appliance/crosscutting/logs"
	"github.com/dumacp/go-ignition/appliance/services/messages"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	//ClientID pubsub id
	ClientID = "ignition"
	//TopicAppliance prefix topic
	TopicAppliance = "appliance/ignition"
	//TopicEvents events topic
	TopicEvents = TopicAppliance + "/events"
)

// //Gateway interface
// type Gateway interface {
// 	Receive(ctx actor.Context)
// 	// Publish(topic string, msg []byte)
// }

type pubsubActor struct {
	ctx           actor.Context
	behavior      actor.Behavior
	state         messages.StatusResponse_StateType
	client        mqtt.Client
	subscriptions map[string]func(msg []byte)
}

var instance *pubsubActor
var once sync.Once

//getInstance create pubsub Gateway
func getInstance() *pubsubActor {

	once.Do(func() {
		instance = &pubsubActor{}
		instance.subscriptions = make(map[string]func(msg []byte))
		ctx := actor.EmptyRootContext
		props := actor.PropsFromFunc(instance.Receive)
		_, err := ctx.SpawnNamed(props, "pubsub-actor")
		if err != nil {
			logs.LogError.Panic(err)
		}
	})
	return instance
}

//Init init pubsub instance
func Init() error {
	if getInstance() == nil {
		return fmt.Errorf("error instance")
	}
	return nil
}

type publishMSG struct {
	topic string
	msg   []byte
}

//Publish function to publish messages in pubsub gateway
func Publish(topic string, msg []byte) {
	instance := getInstance()
	instance.ctx.Send(instance.ctx.Self(), &publishMSG{topic: topic, msg: msg})
}

//Subscribe subscribe to topics
func Subscribe(topic string, callback func(msg []byte)) {
	instance := getInstance()
	instance.subscriptions[topic] = callback
	if !instance.client.IsConnected() {
		instance.ctx.PoisonFuture(instance.ctx.Self()).Wait()
		return
	}
	subscribe(topic, callback)
}

func subscribe(topic string, callback func(msg []byte)) error {
	handler := func(client mqtt.Client, m mqtt.Message) {
		m.Ack()
		callback(m.Payload())
	}
	if tk := instance.client.Subscribe(topic, 1, handler); !tk.WaitTimeout(3 * time.Second) {
		if err := tk.Error(); err != nil {
			return err
		}
	}
	return nil
}

//Receive function
func (ps *pubsubActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case *actor.Started:
		logs.LogInfo.Printf("Starting, actor, pid: %v\n", ctx.Self())
		ps.client = client()
		if err := connect(ps.client); err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}
		for k, v := range ps.subscriptions {
			subscribe(k, v)
		}
	case *actor.Stopping:
		ps.client.Disconnect(600)
		logs.LogError.Println("Stopping, actor is about to shut down")
	case *actor.Stopped:
		logs.LogError.Println("Stopped, actor and its children are stopped")
	case *actor.Restarting:
		logs.LogError.Println("Restarting, actor is about to restart")
	default:
		ps.behavior.Receive(ctx)
	}
}

func (ps *pubsubActor) Started(ctx actor.Context) {
	ps.ctx = ctx
	switch ctx.Message().(type) {
	}
}

func (ps *pubsubActor) Stopped(ctx actor.Context) {
}

func client() mqtt.Client {
	opt := mqtt.NewClientOptions()
	// opt.SetAutoReconnect(true)
	opt.SetClientID(fmt.Sprintf("%s-%d", ClientID, time.Now().Unix()))
	opt.SetKeepAlive(30 * time.Second)
	opt.SetConnectRetryInterval(10 * time.Second)
	client := mqtt.NewClient(opt)
	return client
}

func connect(c mqtt.Client) error {
	tk := c.Connect()
	if tk.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("connect wait error")
	}
	if err := tk.Error(); err != nil {
		return err
	}
	return nil
}
