package main

import "github.com/asynkron/protoactor-go/actor"

type listenDemo struct{}

func (l *listenDemo) Receive(ctx actor.Context) {}
