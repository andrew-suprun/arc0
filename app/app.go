package app

import (
	"arch/lifecycle"
	"arch/msg"
	"log"
	"sort"
	"time"
)

type appModel struct {
	paths       []string
	lc          *lifecycle.Lifecycle
	uiIn        chan<- any
	uiOut       <-chan any
	fsIn        chan<- any
	fsOut       <-chan any
	fsScanState chan []msg.ScanState
	infos       msg.ArchiveInfo
	scanned     int
	scanStates  []msg.ScanState
}

func Run(
	paths []string,
	lc *lifecycle.Lifecycle,
	uiIn chan<- any,
	uiOut <-chan any,
	fsIn chan<- any,
	fsOut <-chan any,
	fsScanState chan []msg.ScanState,
) {
	app := &appModel{
		paths:       paths,
		lc:          lc,
		uiIn:        uiIn,
		uiOut:       uiOut,
		fsIn:        fsIn,
		fsOut:       fsOut,
		fsScanState: fsScanState,
	}
	app.scanStates = make([]msg.ScanState, len(paths))
	for i, path := range paths {
		app.scanStates[i] = msg.ScanState{Base: path}
	}
	fsScanState <- make([]msg.ScanState, len(paths))
	go app.run()
	go app.updateScanStats()
}

func (app *appModel) run() {
	for i, path := range app.paths {
		app.fsIn <- msg.CmdScan{Base: path, Index: i}
	}

	for {
		select {
		case event := <-app.uiOut:
			app.handleUiMessage(event)
		case event := <-app.fsOut:
			app.handleFsMessage(event)
		}
	}
}

type state struct {
}

type fileState int

const (
	initial fileState = iota
)

func (app *appModel) analyze() msg.ArchiveInfo {
	sort.Slice(app.infos, func(i, j int) bool {
		ii := app.infos[i]
		jj := app.infos[j]
		if ii.Hash < jj.Hash {
			return true
		}
		if ii.Hash > jj.Hash {
			return false
		}
		if app.baseIdx(ii.Base) < app.baseIdx(jj.Base) {
			return true
		}
		if app.baseIdx(ii.Base) > app.baseIdx(jj.Base) {
			return false
		}
		return ii.Path < jj.Path
	})

	states := make([]state, len(app.infos))
	start, end := 0, 0
	for start < len(app.infos) {
		for end = start + 1; end < len(app.infos); end++ {
			if app.infos[start].Hash != app.infos[end].Hash {
				break
			}
		}
		app.analyzeForHash(states, start, end)
		start = end
	}

	result := msg.ArchiveInfo{}
	return result
}

func (app *appModel) baseIdx(base string) int {
	for i, path := range app.paths {
		if path == base {
			return i
		}
	}
	return 0
}

func (app *appModel) analyzeForHash(states []state, start, end int) {
	// log.Printf("### %v-%v\n", start, end)
	// for i := start; i < end; i++ {
	// 	log.Printf("###     %v/%v\n", app.infos[i].Base, app.infos[i].Path)

	// }
}

func (app *appModel) handleUiMessage(event any) {
	switch event := event.(type) {
	case msg.CmdQuit:
		app.uiIn <- msg.QuitApp{}
	default:
		log.Panicf("### received unhandled ui message: %#v", event)
	}
}

func (app *appModel) handleFsMessage(event any) {
	switch event := event.(type) {
	case msg.ScanError:
		// TODO

	case msg.ScanDone:
		idx := 0
		for i := range app.scanStates {
			if app.scanStates[i].Base == event.Base {
				idx = i
				break
			}
		}
		scanStates := make([]msg.ScanState, len(app.scanStates))
		copy(scanStates, app.scanStates)
		scanStates = append(scanStates[0:idx], scanStates[idx+1:]...)
		app.scanStates = scanStates

		app.uiIn <- app.scanStates

	case msg.ArchiveInfo:
		app.infos = append(app.infos, event...)
		app.scanned++
		log.Printf("app: ArchInfo: len=%d, scanned=%d", len(app.infos), app.scanned)
		if app.scanned == len(app.paths) {
			app.uiIn <- app.analyze()
		}

	default:
		log.Panicf("### received unhandled fs message: %#v", event)
	}
}

func (app *appModel) updateScanStats() {
	for !app.lc.ShoudStop() {
		time.Sleep(16 * time.Microsecond)
		event := <-app.fsScanState
		app.fsScanState <- event
		app.uiIn <- event
	}
}
