package main

import (
	"log"
	"os"
	"scanner/fs"
	"scanner/lifecycle"
	"scanner/ui"
)

func main() {
	f, err := os.OpenFile("debug.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	log.SetOutput(f)

	uiIn := make(chan any)
	uiOut := make(chan any)
	m := mainModel{uiIn: uiIn, uiOut: uiOut}
	go m.mainLoop()
	ui.Run(uiIn, uiOut)
}

type mainModel struct {
	uiIn    chan<- any
	uiOut   <-chan any
	scanOut <-chan any
}

func (m *mainModel) mainLoop() {
	lc := lifecycle.New()
	scanOut := make(chan any)
	m.scanOut = scanOut
	for _, path := range os.Args[1:] {
		go fs.Scan(lc, path, scanOut)
	}
	for {
		select {
		case msg := <-m.uiOut:
			m.handleUiMessage(msg)
		case msg := <-scanOut:
			m.handleScanMessage(msg)
		}
	}
}

func (m *mainModel) handleUiMessage(msg any) {
	log.Println("arch: ui msg =", msg)
}

func (m *mainModel) handleScanMessage(msg any) {
	switch msg := msg.(type) {
	case fs.ScanStat:
		m.uiIn <- msg
	case fs.ScanFileResult:
		log.Println("arch: scan msg =", msg)
	default:
		log.Panicf("### received unhandled scan message: %#v", msg)
	}
}
