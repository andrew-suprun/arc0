package app

import (
	"arch/lifecycle"
	"arch/msg"
	"log"
	"sort"
)

type appModel struct {
	lc      *lifecycle.Lifecycle
	uiIn    chan<- any
	uiOut   <-chan any
	fsIn    chan<- any
	fsOut   <-chan any
	paths   []string
	infos   msg.ArchiveInfo
	scanned int
}

func Run(paths []string, lc *lifecycle.Lifecycle, uiIn chan<- any, uiOut <-chan any, fsIn chan<- any, fsOut <-chan any) {
	app := &appModel{lc: lc, uiIn: uiIn, uiOut: uiOut, fsIn: fsIn, fsOut: fsOut, paths: paths}
	go app.run()
}

func (app *appModel) run() {
	for _, path := range app.paths {
		app.uiIn <- msg.CmdScan{Base: path}
		app.fsIn <- msg.CmdScan{Base: path}
	}

	for {
		if app.lc.ShoudStop() {
			break
		}
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
		if ii.Base < jj.Base {
			return true
		}
		if ii.Base > jj.Base {
			return false
		}
		return ii.Path < jj.Path
	})
	log.Println(app.infos)
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

func (app *appModel) analyzeForHash(states []state, start, end int) {
	// log.Printf("### %v-%v\n", start, end)
	// for i := start; i < end; i++ {
	// 	log.Printf("###     %v/%v\n", app.infos[i].Base, app.infos[i].Path)

	// }
}

func (app *appModel) handleUiMessage(event any) {
	switch event := event.(type) {
	case msg.CmdQuit:
		close(app.fsIn)
		app.lc.Stop()
		app.uiIn <- msg.QuitApp{}
	default:
		log.Panicf("### received unhandled ui message: %#v", event)
	}
}

func (app *appModel) handleFsMessage(event any) {
	switch event := event.(type) {
	case msg.ScanStat:
		app.uiIn <- event

	case msg.ArchiveInfo:
		app.infos = append(app.infos, event...)
		app.scanned++
		log.Printf("app: ArchInfo: len=%d, scanned=%d", len(app.infos), app.scanned)
		if app.scanned == len(app.paths) {
			app.uiIn <- app.analyze()
		}
	case msg.ScanDone:
		app.uiIn <- event

	case msg.ScanError:
		// TODO

	default:
		log.Panicf("### received unhandled fs message: %#v", event)
	}
}
