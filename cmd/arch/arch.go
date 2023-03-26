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

	ui.Run(paths, fs)
}
