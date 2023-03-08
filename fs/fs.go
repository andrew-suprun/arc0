package fs

import (
	"arch/lifecycle"
	"arch/msg"
)

type runner struct {
	*lifecycle.Lifecycle
	in        <-chan any
	out       chan<- any
	scanState chan []msg.ScanState
}

func Run(lc *lifecycle.Lifecycle, in <-chan any, out chan<- any, scanState chan []msg.ScanState) {
	r := &runner{Lifecycle: lc, in: in, out: out, scanState: scanState}
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
}

func (r *runner) handleCommand(cmd any) {
	switch cmd := cmd.(type) {
	case msg.CmdScan:
		r.scan(cmd.Base, cmd.Index)
	}
}
