package lifecycle

import (
	"context"
	"sync"
)

type Lifecycle struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New() *Lifecycle {
	ctx, cancel := context.WithCancel(context.Background())
	return &Lifecycle{ctx: ctx, cancel: cancel, wg: sync.WaitGroup{}}
}

func (lc *Lifecycle) Started() {
	lc.wg.Add(1)
}

func (lc *Lifecycle) ShoudStop() bool {
	select {
	case <-lc.ctx.Done():
		return true
	default:
		return false
	}
}

func (lc *Lifecycle) Done() {
	lc.wg.Done()
}

func (lc *Lifecycle) Stop() {
	lc.cancel()
	lc.wg.Wait()
}
