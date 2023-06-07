package actor

import (
	"fmt"
	"runtime/debug"
	"sync"
)

type Handler func(any)

func NewActor(handler Handler) Actor {
	actor := &actor{
		handler: handler,
		Cond:    sync.NewCond(&sync.Mutex{}),
	}
	go run(actor)
	return actor
}

type Actor interface {
	Send(message any)
}

type actor struct {
	handler Handler
	pending []any
	*sync.Cond
}

func run(a *actor) {
	for {
		a.Cond.L.Lock()

		if len(a.pending) == 0 {
			a.Cond.Wait()
			a.Cond.L.Unlock()
			continue
		}

		msg := a.pending[0]
		a.pending = a.pending[1:]

		a.Cond.L.Unlock()
		a.handleMessage(msg)
	}
}

func (a *actor) handleMessage(msg any) {
	defer func() {
		if r := recover(); r != nil {
			a.Cond.L.Lock()
			fmt.Printf("Actor %v panicked: %v\n", a, r)
			fmt.Println(string(debug.Stack()))
			a.Cond.L.Unlock()
		}
	}()
	a.handler(msg)
}

func (a *actor) Send(msg any) {
	a.Cond.L.Lock()
	a.pending = append(a.pending, msg)
	a.Cond.Signal()
	a.Cond.L.Unlock()
}
