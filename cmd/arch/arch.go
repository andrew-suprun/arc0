package main

import (
	"arch/files"
	"arch/files/file_fs"
	"arch/files/mock_fs"
	"arch/ui"
	"log"
	"os"
)

func main() {
	// log.SetFlags(0)

	var fsys files.FS
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

	runner := ui.NewUi(paths, fsys)
	log.Println("main.3")
	runner.Run()
	log.Println("main.4")
}
