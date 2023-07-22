package stream

import (
	"sync"
)

type Stream[T any] struct {
	name     string
	elements []T
	*sync.Cond
}

func NewStream[T any](name string) *Stream[T] {
	return &Stream[T]{
		Cond: sync.NewCond(&sync.Mutex{}),
		name: name,
	}
}

func (s *Stream[T]) Push(msg T) {
	s.Cond.L.Lock()
	s.elements = append(s.elements, msg)
	s.Cond.Signal()
	s.Cond.L.Unlock()
}

func (s *Stream[T]) Pull() []T {
	for {
		s.Cond.L.Lock()
		if len(s.elements) == 0 {
			s.Cond.Wait()
			s.Cond.L.Unlock()
			continue
		} else {
			msgs := s.elements
			s.elements = []T{}
			s.Cond.L.Unlock()
			return msgs
		}
	}
}

func (s *Stream[T]) TryPull() []T {
	s.Cond.L.Lock()
	msgs := s.elements
	s.elements = []T{}
	s.Cond.L.Unlock()
	return msgs
}
