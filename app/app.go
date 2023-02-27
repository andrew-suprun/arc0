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
	stats   []scanStats
}

type scanStats struct {
	metas  msg.ScanMetas
	byName map[string]*msg.FileMeta
	byHash map[string]msg.ScanMetas
}

func Run(paths []string, uiIn chan<- any, uiOut <-chan any, fsIn chan<- any, fsOut <-chan any) {
	app := &appModel{uiIn: uiIn, uiOut: uiOut, fsIn: fsIn, fsOut: fsOut}

	app.stats = make([]scanStats, len(paths))
	for i, path := range paths {
		app.stats[i] = scanStats{
			metas:  msg.ScanMetas{Base: path},
			byName: map[string]*msg.FileMeta{},
			byHash: map[string]msg.ScanMetas{},
		}
		app.uiIn <- msg.CmdScan{Base: path}
		app.fsIn <- msg.CmdScan{Base: path}
	}

	for app.scanned < len(paths) {
		log.Println("Run: wait for event")
		select {
		case event := <-app.uiOut:
			app.handleUiMessage(event)
		case event := <-app.fsOut:
			app.handleFsMessage(event)
		}
	}
	app.analyze()

	app.uiIn <- msg.QuitApp{}
}

func (app *appModel) analyze() {
	hashes := make(map[string]struct{})
	for _, stats := range app.stats {
		for _, fileMeta := range stats.metas.Metas {
			hashes[fileMeta.Hash] = struct{}{}
		}
	}

	result := msg.Analysis{}
	for hash := range hashes {
		byHash := make([]msg.ScanMetas, len(app.stats))
		extraFiles := false
		for i, stats := range app.stats {
			byHash[i] = stats.byHash[hash]
			if len(byHash[i].Metas) > len(byHash[0].Metas) {
				extraFiles = true
			}
		}
		if extraFiles {
			hashResult := []msg.ScanMetas{byHash[0]}
			for i := 1; i < len(byHash); i++ {
				if len(byHash[i].Metas) > len(byHash[0].Metas) {
					hashResult = append(hashResult, byHash[i])
				}
			}
			result = append(result, hashResult)
		}
	}
	log.Println("analyze 1:", result)
	app.uiIn <- result
	log.Println("analyze 2")
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
	log.Printf("app.handleFsMessage: %#v\n", event)
	switch event := event.(type) {
	case msg.ScanStat:
		app.uiIn <- event

	case msg.ScanMetas:
		log.Printf("app.ScanMetas\n")
		for i := range app.stats {
			stats := &app.stats[i]
			if stats.metas.Base == event.Base {
				stats.metas.Metas = event.Metas
				for _, meta := range event.Metas {
					stats.byName[meta.Path] = meta

					m := append(stats.byHash[meta.Hash].Metas, meta)
					stats.byHash[meta.Hash] = msg.ScanMetas{Base: meta.Base, Metas: m}
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
