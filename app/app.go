package app

import (
	"arch/msg"
	"log"
)

type appModel struct {
	uiIn    chan<- any
	uiOut   <-chan any
	fsIn    chan<- any
	fsOut   <-chan any
	scanned int
	stats   map[string]*scanStats // key: base
	source  string
}

type scanStats struct {
	metas  msg.FileMetas
	byName map[string]*msg.FileMeta
	byHash map[string]msg.FileMetas
}

func Run(paths []string, uiIn chan<- any, uiOut <-chan any, fsIn chan<- any, fsOut <-chan any) {
	app := &appModel{source: paths[0], uiIn: uiIn, uiOut: uiOut, fsIn: fsIn, fsOut: fsOut}

	app.stats = make(map[string]*scanStats)
	for _, path := range paths {
		app.stats[path] = &scanStats{
			metas:  msg.FileMetas{},
			byName: map[string]*msg.FileMeta{},
			byHash: map[string]msg.FileMetas{},
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

	app.uiIn <- app.analyze()
}

func (app *appModel) analyze() msg.Analysis {
	hashes := make(map[string]struct{})
	for _, stats := range app.stats {
		for _, fileMeta := range stats.metas {
			hashes[fileMeta.Hash] = struct{}{}
		}
	}

	result := msg.Analysis{}
	for hash := range hashes {
		forHash := make(map[string]msg.FileMetas, len(app.stats))
		for base, stats := range app.stats {
			byHash := stats.byHash[hash]
			if len(byHash) > 0 {
				forHash[base] = stats.byHash[hash]
			}
		}
		extraFiles := false
		for base := range app.stats {
			if len(forHash[app.source]) < len(forHash[base]) {
				extraFiles = true
			}
		}
		if extraFiles {
			result[hash] = forHash
		}
	}
	return result
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
		stats := app.stats[event.Base]
		stats.metas = event.Metas
		for _, meta := range event.Metas {
			stats.byName[meta.Path] = meta
			stats.byHash[meta.Hash] = append(stats.byHash[meta.Hash], meta)
		}

		app.scanned++
		app.uiIn <- msg.ScanDone{Base: event.Base}

	case msg.ScanError:
		// TODO

	default:
		log.Panicf("### received unhandled fs message: %#v", event)
	}
}
