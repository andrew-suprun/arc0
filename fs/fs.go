package fs

import (
	"arch/lifecycle"
	"arch/msg"
)

type runner struct {
	*lifecycle.Lifecycle
	in  <-chan any
	out chan<- any
}

func Run(lc *lifecycle.Lifecycle, in <-chan any, out chan<- any) {
	r := &runner{Lifecycle: lifecycle.New(), in: in, out: out}
	go r.run()
}

func (r *runner) run() {
	for {
		cmd, ok := <-r.in
		if !ok {
			break
		}
		go r.handleCommand(cmd)
	}
	r.Stop()
}

func (r *runner) handleCommand(cmd any) {
	switch cmd := cmd.(type) {
	case msg.CmdScan:
		r.scan(cmd.Base)
	}
}
