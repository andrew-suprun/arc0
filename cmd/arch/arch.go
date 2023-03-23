package main

import (
	"arch/app"
	"arch/files/file_fs"
	"arch/files/mock_fs"
	"arch/tcell"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)

	var fsys app.FS
	var paths []string

	if len(os.Args) >= 1 && os.Args[1] == "-sim" {
		fsys = mock_fs.NewFs()
		paths = os.Args[2:]
	} else {
		fsys = file_fs.NewFs()
		paths = os.Args[1:]
	}

	for _, path := range paths {
		if !fsys.IsValid(path) {
			log.Printf("Invalid path: %v", path)
			return
		}
	}

	tcell.Run(app.NewApp(paths, fsys))
}
