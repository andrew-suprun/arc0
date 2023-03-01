package app

import (
	"arch/msg"
	"log"
	"sync"
	"testing"
)

func TestAnalyze(t *testing.T) {
	log.SetFlags(0)
	wg := sync.WaitGroup{}
	wg.Add(1)
	uiIn := make(chan any)
	uiOut := make(chan any)

	fsIn := make(chan any)
	fsOut := make(chan any)

	go func() {
		for {
			select {
			case event := <-fsIn:
				go handleFsEvent(t, event, fsOut)
			case event := <-uiIn:
				go handleUiEvent(t, &wg, event, uiOut)
			}
		}
	}()

	Run([]string{"source", "copy1", "copy2"}, uiIn, uiOut, fsIn, fsOut)
	wg.Wait()
}

func handleFsEvent(t *testing.T, event any, out chan any) {
	switch event := event.(type) {
	case msg.CmdScan:
		out <- scanResults(event.Base)
	default:
		log.Printf("fsIn: %#v\n", event)
	}
}

func handleUiEvent(t *testing.T, wg *sync.WaitGroup, event any, out chan any) {
	switch event := event.(type) {
	case msg.CmdScan:
	case msg.ScanDone:
	case msg.Analysis:
		if len(event) != 3 || len(event["cccc"]) != 3 || len(event["aaa1"]) != 1 || len(event["aaa2"]) != 1 {
			t.Fail()
		}
		wg.Done()
	default:
		log.Printf("uiIn: %#v\n", event)
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
