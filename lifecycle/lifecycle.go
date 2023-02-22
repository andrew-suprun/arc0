package lifecycle

import (
	"context"
)

type Lifecycle struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func New() *Lifecycle {
	ctx, cancel := context.WithCancel(context.Background())
	return &Lifecycle{ctx: ctx, cancel: cancel}
}

func (lc *Lifecycle) ShoudStop() bool {
	select {
	case <-lc.ctx.Done():
		return true
	default:
		return false
	}
}

func (lc *Lifecycle) Stop() {
	lc.cancel()
}
