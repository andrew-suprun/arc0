package main

import (
	"arch/app"
	"arch/fs"
	"arch/ui"
	"fmt"
	"os"
)

func main() {
	paths := os.Args[1:]
	for _, path := range paths {
		_, err := os.Stat(path)
		if err != nil {
			fmt.Printf("Invalid path: %v", path)
			return
		}
	}

	uiIn := make(chan any)
	uiOut := make(chan any)

	fsIn := make(chan any)
	fsOut := make(chan any)

	go app.Run(paths, uiIn, uiOut, fsIn, fsOut)
	go fs.Run(fsIn, fsOut)
	ui.Run(uiIn, uiOut)
}
