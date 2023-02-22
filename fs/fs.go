package fs

import (
	"scanner/api"
	"scanner/lifecycle"
)

type runner struct {
	in  <-chan any
	out chan<- any
	*lifecycle.Lifecycle
}

func Run(in <-chan any, out chan<- any) {
	r := &runner{in: in, out: out, Lifecycle: lifecycle.New()}
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
	r.Lifecycle.Stop()
}

func (r *runner) handleCommand(cmd any) {
	switch cmd := cmd.(type) {
	case api.CmdScan:
		r.scan(cmd.Base)
	}
}
