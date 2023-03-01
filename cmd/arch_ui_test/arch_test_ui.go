package main

import (
	"arch/app"
	"arch/msg"
	"arch/ui"
)

func main() {
	uiIn := make(chan any)
	uiOut := make(chan any)

	fsIn := make(chan any)
	fsOut := make(chan any)

	go app.Run([]string{"source", "copy1", "copy2"}, uiIn, uiOut, fsIn, fsOut)
	go fsRun(fsIn, fsOut)
	ui.Run(uiIn, uiOut)
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

func scanResults(base string) msg.ScanMetas {
	switch base {
	case "source":
		return msg.ScanMetas{
			Base:  base,
			Metas: source,
		}
	case "copy1":
		return msg.ScanMetas{
			Base:  base,
			Metas: copy1,
		}
	case "copy2":
		return msg.ScanMetas{
			Base:  base,
			Metas: copy2,
		}
	}
	return msg.ScanMetas{}
}

var source = msg.FileMetas{
	{Path: "a", Size: 1, Hash: "aaaa"},
	{Path: "b1", Size: 2, Hash: "bbbb"},
	{Path: "b2", Size: 2, Hash: "bbbb"},
	{Path: "c", Size: 3, Hash: "cccc"},
}

var copy1 = msg.FileMetas{
	{Path: "a", Size: 1, Hash: "aaa1"},
	{Path: "b", Size: 2, Hash: "bbbb"},
	{Path: "c", Size: 3, Hash: "cccc"},
	{Path: "d", Size: 3, Hash: "cccc"},
}

var copy2 = msg.FileMetas{
	{Path: "a", Size: 1, Hash: "aaa2"},
	{Path: "b", Size: 2, Hash: "bbbb"},
	{Path: "c1", Size: 3, Hash: "cccc"},
	{Path: "c2", Size: 3, Hash: "cccc"},
}
