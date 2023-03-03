package main

import (
	"arch/app"
	"arch/fs"
	"arch/lifecycle"
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

	app.Run(paths, lc, uiIn, uiOut, fsIn, fsOut)
	fs.Run(lc, fsIn, fsOut)
	ui.Run(uiIn, uiOut)
}
