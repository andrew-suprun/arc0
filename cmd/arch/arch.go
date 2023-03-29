package main

import (
	"arch/app"
	"arch/files"
	"arch/files/file_fs"
	"arch/files/mock_fs"
	"arch/ui/tcell"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)

	var fs files.FS
	var paths []string

	if len(os.Args) >= 1 && os.Args[1] == "-sim" {
		fs = mock_fs.NewFs()
		paths = os.Args[2:]
	} else {
		fs = file_fs.NewFs()
		paths = os.Args[1:]
	}

	for _, path := range paths {
		if !fs.IsValid(path) {
			log.Printf("Invalid path: %v", path)
			return
		}
	}

	renderer, err := tcell.NewRenderer()
	if err != nil {
		log.Printf("Failed to open terminal: %#v", err)
		return
	}

	app.Run(paths, fs, renderer)
}
