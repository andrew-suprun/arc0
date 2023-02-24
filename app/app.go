package app

import (
	"arch/msg"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type mainModel struct {
	uiIn    chan<- any
	uiOut   <-chan any
	fsIn    chan<- any
	fsOut   <-chan any
	toScan  int
	scanned int
}

func Run(uiIn chan<- any, uiOut <-chan any, fsIn chan<- any, fsOut <-chan any) {
	a := &mainModel{uiIn: uiIn, uiOut: uiOut, fsIn: fsIn, fsOut: fsOut}
	paths := os.Args[1:]
	for _, path := range paths {
		a.uiIn <- msg.CmdScan{Base: path}
		a.fsIn <- msg.CmdScan{Base: path}
	}
	a.toScan = len(paths)
	for a.scanned < a.toScan {
		select {
		case event := <-a.uiOut:
			a.handleUiMessage(event)
		case event := <-a.fsOut:
			a.handleFsMessage(event)
		}
	}
	a.uiIn <- tea.Quit()
}

func (a *mainModel) handleUiMessage(event any) {
	log.Printf("arch: ui event = %#v", event)
	switch event := event.(type) {
	case msg.CmdQuit:
		close(a.fsIn)
	default:
		log.Panicf("### received unhandled ui message: %#v", event)
	}
}

func (a *mainModel) handleFsMessage(event any) {
	switch event := event.(type) {
	case msg.ScanStat:
		a.uiIn <- event

	case msg.ScanDone:
		a.scanned++
		a.uiIn <- event

	case msg.FileMeta:
		// TODO

	case msg.ScanError:
		// TODO

	default:
		log.Panicf("### received unhandled fs message: %#v", event)
	}
}
