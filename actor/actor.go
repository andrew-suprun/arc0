package actor

import (
	"sync"
)

type Actor[T any] interface {
	Send(message T)
}

type Handler[T any] func(T) bool

func NewActor[T any](handler Handler[T]) Actor[T] {
	actor := &actor[T]{
		handler: handler,
		Cond:    sync.NewCond(&sync.Mutex{}),
	}
	go run(actor)
	return actor
}

type actor[T any] struct {
	handler Handler[T]
	pending []T
	*sync.Cond
}

func (a *actor[T]) Send(msg T) {
	a.Cond.L.Lock()
	a.pending = append(a.pending, msg)
	a.Cond.Signal()
	a.Cond.L.Unlock()
}

func run[T any](a *actor[T]) {
	running := true
	for running {
		a.Cond.L.Lock()

		if len(a.pending) == 0 {
			a.Cond.Wait()
			a.Cond.L.Unlock()
			continue
		}

		msg := a.pending[0]
		a.pending = a.pending[1:]

		a.Cond.L.Unlock()
		running = a.handler(msg)
	}
}
