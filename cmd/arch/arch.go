package main

import (
	"arch/device/tcell"
	"arch/files"
	"arch/files/file_fs"
	"arch/files/mock2_fs"
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
		paths = []string{"origin", "copy 1", "copy 2"}
	} else if len(os.Args) >= 1 && os.Args[1] == "-sim2" {
		fs = mock2_fs.NewFs()
		paths = []string{"origin", "copy 1", "copy 2"}
	} else {
		fs = file_fs.NewFs()
		paths = os.Args[1:]
	}

	for i, path := range paths {
		var err error
		path, err = fs.Abs(path)
		if err != nil {
			log.Printf("Invalid path: %v", path)
			return
		}
		paths[i] = path
	}

	device, err := tcell.NewDevice()
	if err != nil {
		log.Printf("Failed to open terminal: %#v", err)
		return
	}

	ui.Run(device, fs, paths)
}
