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

	for _, path := range paths {
		if !fs.IsValid(path) {
			log.Printf("Invalid path: %v", path)
			return
		}
	}

	device, err := tcell.NewDevice()
	if err != nil {
		log.Printf("Failed to open terminal: %#v", err)
		return
	}

	events := make(chan any)

	go func() {
		for {
			events <- device.PollEvent()
		}
	}()

	for _, archive := range paths {
		go func(archive string) {
			for ev := range fs.Scan(archive) {
				events <- ev
			}
		}(archive)
	}

	ui.Run(device, events, paths)

	fs.Stop()
}
