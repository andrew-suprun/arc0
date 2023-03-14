package app

import (
	"arch/lifecycle"
	"arch/msg"
	"arch/ui"
	"log"
	"path/filepath"
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
	scanStarted time.Time
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

// TODO: redo
func (app *appModel) analyze() ui.Archive {
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

	result := ui.Archive{}
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

	case msg.ArchiveInfo:
		app.infos = append(app.infos, event...)
		app.scanned++
		if app.scanned == len(app.paths) {
			app.uiIn <- app.analyze()
		}

	default:
		log.Panicf("### received unhandled fs message: %#v", event)
	}
}

var nilTime time.Time

func (app *appModel) updateScanStats() {
	for !app.lc.ShoudStop() {
		time.Sleep(16 * time.Microsecond)
		event := <-app.fsScanState
		stats := make(ui.ScanStates, 0, len(event))
		if app.scanStarted == nilTime {
			app.scanStarted = time.Now()
		}
		for _, state := range event {
			if state.Path == "" {
				continue
			}
			etaProgress := float64(state.TotalToHash) / float64(state.TotalHashed)
			hashed := state.TotalSize - state.TotalToHash + state.TotalHashed
			dur := time.Since(app.scanStarted)
			eta := app.scanStarted.Add(time.Duration(float64(dur) * etaProgress))

			stats = append(stats, ui.ScanState{
				Archive:       state.Base,
				Folder:        filepath.Dir(state.Path),
				File:          filepath.Base(state.Path),
				ETA:           eta,
				RemainingTime: time.Until(eta),
				Progress:      float64(hashed) / float64(state.TotalSize),
			})
		}

		app.fsScanState <- event
		if app.scanned == len(app.paths) {
			app.uiIn <- ui.ScanStates{}
			break
		}
		app.uiIn <- stats
	}
}
