package main

import (
	"arch/app"
	"arch/lifecycle"
	"arch/msg"
	"arch/ui/tcell"
	"log"
)

func main() {
	uiIn := make(chan any)
	uiOut := make(chan any)

	fsIn := make(chan any)
	fsOut := make(chan any)
	fsScanState := make(chan []msg.ScanState, 1)
	lc := lifecycle.New()

	app.Run([]string{"source", "copy1", "copy2"}, lc, uiIn, uiOut, fsIn, fsOut, fsScanState)
	go fsRun(fsIn, fsOut)
	tcell.Run(lc, uiIn, uiOut)
}

func fsRun(in <-chan any, out chan<- any) {
	for {
		event, ok := <-in
		if !ok {
			break
		}
		switch event := event.(type) {
		case msg.CmdScan:
			go func() {
				out <- scanResults(event.Base)
			}()
		}
	}
}

func scanResults(base string) msg.ArchiveInfo {
	log.Println("### scanResults: base =", base)
	switch base {
	case "source":
		return msg.ArchiveInfo{
			{Base: "source", Path: "a", Size: 1, Hash: "aaaa"},
			{Base: "source", Path: "b1", Size: 2, Hash: "bbbb"},
			{Base: "source", Path: "b2", Size: 2, Hash: "bbbb"},
			{Base: "source", Path: "c", Size: 3, Hash: "cccc"},
		}

	case "copy1":
		return msg.ArchiveInfo{
			{Base: "copy1", Path: "a", Size: 1, Hash: "aaa1"},
			{Base: "copy1", Path: "b", Size: 2, Hash: "bbbb"},
			{Base: "copy1", Path: "c", Size: 3, Hash: "cccc"},
			{Base: "copy1", Path: "d", Size: 3, Hash: "cccc"},
		}
	case "copy2":
		return msg.ArchiveInfo{
			{Base: "copy2", Path: "a", Size: 1, Hash: "aaa2"},
			{Base: "copy2", Path: "b", Size: 2, Hash: "bbbb"},
			{Base: "copy2", Path: "c1", Size: 3, Hash: "cccc"},
			{Base: "copy2", Path: "c2", Size: 3, Hash: "cccc"},
		}
	}
	return msg.ArchiveInfo{}
}
