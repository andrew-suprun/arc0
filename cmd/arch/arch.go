package main

import (
	"arch/controller"
	"arch/files/file_fs"
	"arch/files/mock_fs"
	"arch/lifecycle"
	m "arch/model"
	"arch/renderer/tcell"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	logFile, err := os.Create("log.log")
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	var paths []m.Root
	if len(os.Args) >= 1 && (os.Args[1] == "-sim" || os.Args[1] == "-sim2") {
		paths = []m.Root{"origin", "copy 1", "copy 2"}
	} else {
		paths = make([]m.Root, len(os.Args)-1)
		for i, path := range os.Args[1:] {
			path, err := file_fs.AbsPath(path)
			paths[i] = m.Root(path)
			if err != nil {
				log.Panicf("Failed to scan archives: %#v", err)
			}
		}
	}

	lc := lifecycle.New()
	events := make(m.EventChan, 256)
	renderer, err := tcell.NewRenderer(events)
	if err != nil {
		log.Printf("Failed to open terminal: %#v", err)
		return
	}

	var fs m.FS

	if len(os.Args) >= 1 && os.Args[1] == "-sim" {
		fs = mock_fs.NewFs(events)
		mock_fs.Scan = true
	} else if len(os.Args) >= 1 && os.Args[1] == "-sim2" {
		fs = mock_fs.NewFs(events)
	} else {
		fs = file_fs.NewFs(events, lc)
	}

	controller.Run(fs, renderer, events, paths)

	lc.Stop()
	renderer.Stop()
}
