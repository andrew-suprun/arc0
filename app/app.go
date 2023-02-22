package app

import (
	"log"
	"os"
	"scanner/api"
	"scanner/fs"
	"scanner/ui"

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

func Run() {
	uiIn := make(chan any)
	uiOut := make(chan any)

	fsIn := make(chan any)
	fsOut := make(chan any)

	a := &mainModel{uiIn: uiIn, uiOut: uiOut, fsIn: fsIn, fsOut: fsOut}
	go a.run()
	go fs.Run(fsIn, fsOut)
	ui.Run(uiIn, uiOut)
}

func (a *mainModel) run() {
	paths := os.Args[1:]
	for _, path := range paths {
		a.uiIn <- api.CmdScan{Base: path}
		a.fsIn <- api.CmdScan{Base: path}
	}
	a.toScan = len(paths)
	for a.scanned < a.toScan {
		select {
		case msg := <-a.uiOut:
			a.handleUiMessage(msg)
		case msg := <-a.fsOut:
			a.handleFsMessage(msg)
		}
	}
	a.uiIn <- tea.Quit()
}

func (a *mainModel) handleUiMessage(msg any) {
	log.Println("arch: ui msg =", msg)
	switch msg := msg.(type) {
	case api.CmdQuit:
		close(a.fsIn)
	default:
		log.Panicf("### received unhandled ui message: %#v", msg)
	}
}

func (a *mainModel) handleFsMessage(msg any) {
	switch msg := msg.(type) {
	case api.ScanStat:
		a.uiIn <- msg

	case api.ScanDone:
		a.scanned++
		a.uiIn <- msg

	case api.FileMeta:

	default:
		log.Panicf("### received unhandled fs message: %#v", msg)
	}
}
