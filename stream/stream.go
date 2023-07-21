package stream

import (
	"sync"
)

type Stream[T any] interface {
	Push(message T)
	Pull() T
	PullAll() []T
}

func NewStream[T any](name string) Stream[T] {
	stream := &stream[T]{
		Cond: sync.NewCond(&sync.Mutex{}),
		name: name,
	}
	return stream
}

type stream[T any] struct {
	name     string
	elements []T
	*sync.Cond
}

func (s *stream[T]) Push(msg T) {
	s.Cond.L.Lock()
	s.elements = append(s.elements, msg)
	s.Cond.Signal()
	s.Cond.L.Unlock()
}

func (s *stream[T]) Pull() T {
	for {
		s.Cond.L.Lock()
		if len(s.elements) == 0 {
			s.Cond.Wait()
			s.Cond.L.Unlock()
			continue
		}
		msg := s.elements[0]
		s.elements = s.elements[1:]

		s.Cond.L.Unlock()
		return msg
	}
}

func (s *stream[T]) PullAll() []T {
	s.Cond.L.Lock()
	msgs := s.elements
	s.elements = []T{}
	s.Cond.L.Unlock()
	return msgs
}
