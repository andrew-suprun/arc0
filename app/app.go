package app

import (
	"arch/msg"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

type appModel struct {
	uiIn    chan<- any
	uiOut   <-chan any
	fsIn    chan<- any
	fsOut   <-chan any
	scanned int
	stats   []scanStats
}

type scanStats struct {
	base   string
	byName map[string]*fileState
	byHash map[string]*fileState
}

type fileState struct {
	state
	*msg.FileMeta
}

type state int

const (
	initial state = iota
)

func Run(paths []string, uiIn chan<- any, uiOut <-chan any, fsIn chan<- any, fsOut <-chan any) {
	app := &appModel{uiIn: uiIn, uiOut: uiOut, fsIn: fsIn, fsOut: fsOut}

	app.stats = make([]scanStats, len(paths))
	for i, path := range paths {
		app.stats[i] = scanStats{
			base:   path,
			byName: map[string]*fileState{},
			byHash: map[string]*fileState{},
		}
		app.uiIn <- msg.CmdScan{Base: path}
		app.fsIn <- msg.CmdScan{Base: path}
	}

	for app.scanned < len(paths) {
		select {
		case event := <-app.uiOut:
			app.handleUiMessage(event)
		case event := <-app.fsOut:
			app.handleFsMessage(event)
		}
	}
	app.analyze()
	app.uiIn <- tea.Quit()
}

func (app *appModel) analyze() {
	source := app.stats[0]
	targets := app.stats[1:]
	log.Println("### source", source.base)
	for _, t := range targets {
		log.Println("### target", t.base)
	}
}

func (app *appModel) handleUiMessage(event any) {
	switch event := event.(type) {
	case msg.CmdQuit:
		close(app.fsIn)
	default:
		log.Panicf("### received unhandled ui message: %#v", event)
	}
}

func (app *appModel) handleFsMessage(event any) {
	switch event := event.(type) {
	case msg.ScanStat:
		app.uiIn <- event

	case msg.ScanMetas:
		for i := range app.stats {
			stats := &app.stats[i]
			if stats.base == event.Base {
				for _, meta := range event.Metas {
					s := &fileState{FileMeta: meta}
					stats.byName[meta.Path] = s
					stats.byHash[meta.Hash] = s
				}
				break
			}
		}
		app.scanned++
		app.uiIn <- msg.ScanDone{Base: event.Base}

	case msg.ScanError:
		// TODO

	default:
		log.Panicf("### received unhandled fs message: %#v", event)
	}
}
