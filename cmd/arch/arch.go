package main

import (
	"arch/app"
	"arch/fs"
	"arch/ui"
)

func main() {
	uiIn := make(chan any)
	uiOut := make(chan any)

	fsIn := make(chan any)
	fsOut := make(chan any)

	go app.Run(uiIn, uiOut, fsIn, fsOut)
	go fs.Run(fsIn, fsOut)
	ui.Run(uiIn, uiOut)
}
