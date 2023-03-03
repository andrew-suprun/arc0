package app

import (
	"arch/lifecycle"
	"log"
	"testing"
)

func TestAnalyze(t *testing.T) {
	log.SetFlags(0)
	uiIn := make(chan any)
	uiOut := make(chan any)

	fsIn := make(chan any)
	fsOut := make(chan any)

	lc := lifecycle.New()

	go func() {
		for {
			select {
			case event := <-fsIn:
				go handleFsEvent(t, event, fsOut)
			case event := <-uiIn:
				go handleUiEvent(t, event, uiOut, lc)
			}
		}
	}()

	Run([]string{"source", "copy1", "copy2"}, lc, uiIn, uiOut, fsIn, fsOut)
}

func handleFsEvent(t *testing.T, event any, out chan any) {
	// switch event := event.(type) {
	// case msg.CmdScan:
	// 	out <- scanResults(event.Base)
	// default:
	// 	log.Printf("fsIn: %#v\n", event)
	// }
}

func handleUiEvent(t *testing.T, event any, uiIn chan any, lc *lifecycle.Lifecycle) {
	// log.Printf("### ui event %#v\n", event)
	// switch event := event.(type) {
	// case msg.CmdScan:
	// case msg.ScanDone:
	// case msg.ArchiveInfo:
	// 	if len(event) != 3 || len(event["cccc"]) != 3 || len(event["aaa1"]) != 1 || len(event["aaa2"]) != 1 {
	// 		t.Fail()
	// 	}

	// 	lc.Stop()
	// 	uiIn <- msg.CmdQuit{}
	// default:
	// 	log.Printf("uiIn: %#v\n", event)
	// }
}

// func scanResults(base string) msg.ScanMetas {
// 	switch base {
// 	case "source":
// 		return msg.ScanMetas{
// 			Base:  base,
// 			Metas: source,
// 		}
// 	case "copy1":
// 		return msg.ScanMetas{
// 			Base:  base,
// 			Metas: copy1,
// 		}
// 	case "copy2":
// 		return msg.ScanMetas{
// 			Base:  base,
// 			Metas: copy2,
// 		}
// 	}
// 	return msg.ScanMetas{}
// }

// var source = msg.ArchiveInfo{
// 	{Path: "a", Size: 1, Hash: "aaaa"},
// 	{Path: "b1", Size: 2, Hash: "bbbb"},
// 	{Path: "b2", Size: 2, Hash: "bbbb"},
// 	{Path: "c", Size: 3, Hash: "cccc"},
// }

// var copy1 = msg.ArchiveInfo{
// 	{Path: "a", Size: 1, Hash: "aaa1"},
// 	{Path: "b", Size: 2, Hash: "bbbb"},
// 	{Path: "c", Size: 3, Hash: "cccc"},
// 	{Path: "d", Size: 3, Hash: "cccc"},
// }

// var copy2 = msg.ArchiveInfo{
// 	{Path: "a", Size: 1, Hash: "aaa2"},
// 	{Path: "b", Size: 2, Hash: "bbbb"},
// 	{Path: "c1", Size: 3, Hash: "cccc"},
// 	{Path: "c2", Size: 3, Hash: "cccc"},
// }
