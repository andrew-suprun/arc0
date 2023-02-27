package app

import (
	"arch/msg"
	"log"
	"sync"
	"testing"
)

func TestAnalyze(t *testing.T) {
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
				go handleFsEvent(event, fsOut)
			case event := <-uiIn:
				go handleUiEvent(&wg, event, uiOut)
			}
		}
	}()

	Run([]string{"source", "copy1", "copy2"}, uiIn, uiOut, fsIn, fsOut)
	wg.Wait()
}

func handleFsEvent(event any, out chan any) {
	log.Printf("fsIn: %#v\n", event)
	switch event := event.(type) {
	case msg.CmdScan:
		out <- scanResults(event.Base)
	}
}

func handleUiEvent(wg *sync.WaitGroup, event any, out chan any) {
	log.Printf("uiIn: %#v\n", event)
	switch event := event.(type) {
	case msg.QuitApp:
		wg.Done()
	case msg.Analysis:
		log.Println("metas", event)
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

var source = []*msg.FileMeta{
	{Base: "source", Path: "a", Size: 1, Hash: "aaaa"},
	{Base: "source", Path: "b1", Size: 2, Hash: "bbbb"},
	{Base: "source", Path: "b2", Size: 2, Hash: "bbbb"},
	{Base: "source", Path: "c", Size: 3, Hash: "cccc"},
}

var copy1 = []*msg.FileMeta{
	{Base: "copy1", Path: "a", Size: 1, Hash: "aaa1"},
	{Base: "copy1", Path: "b", Size: 2, Hash: "bbbb"},
	{Base: "copy1", Path: "c", Size: 3, Hash: "cccc"},
	{Base: "copy1", Path: "d", Size: 3, Hash: "cccc"},
}

var copy2 = []*msg.FileMeta{
	{Base: "copy2", Path: "a", Size: 1, Hash: "aaa2"},
	{Base: "copy2", Path: "b", Size: 2, Hash: "bbbb"},
	{Base: "copy2", Path: "c1", Size: 3, Hash: "cccc"},
	{Base: "copy2", Path: "c2", Size: 3, Hash: "cccc"},
}
