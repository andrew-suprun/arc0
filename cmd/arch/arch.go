package main

import (
	"arch/app"
	"arch/fs"
	"arch/lifecycle"
	"arch/msg"
	"arch/ui"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	paths := os.Args[1:]
	for _, path := range paths {
		_, err := os.Stat(path)
		if err != nil {
			log.Printf("Invalid path: %v", path)
			return
		}
	}

	lc := lifecycle.New()

	uiIn := make(chan any)
	uiOut := make(chan any)

	fsIn := make(chan any)
	fsOut := make(chan any)
	fsScanState := make(chan []msg.ScanState, 1)

	app.Run(paths, lc, uiIn, uiOut, fsIn, fsOut, fsScanState)
	fs.Run(lc, fsIn, fsOut, fsScanState)
	ui.Run(lc, uiIn, uiOut)
}
