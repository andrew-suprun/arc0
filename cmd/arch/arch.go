package main

import (
	"arch/device"
	"arch/device/tcell"
	"arch/files/file_fs"
	"arch/lifecycle"
	"arch/model"
	"arch/view"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)

	m := model.Model{}

	events := make(model.EventHandler)

	// if len(os.Args) >= 1 && os.Args[1] == "-sim" {
	// 	startMock("origin", events)
	// 	startMock("copy 1", events)
	// 	startMock("copy 2", events)
	// } else if len(os.Args) >= 1 && os.Args[1] == "-sim2" {
	// 	startMock2("origin,", events)
	// 	startMock2("copy 1", events)
	// 	startMock2("copy 2", events)
	// } else {
	lc := lifecycle.Lifecycle{}
	m.ArchivePaths = os.Args[1:]
	for _, path := range os.Args[1:] {
		fsys, err := file_fs.NewFs(path, events, &lc)
		if err != nil {
			log.Panicf("Failed to scan archive %s: %#v", path, err)
		}
		fsys.Scan()
	}
	// }

	d, err := tcell.NewDevice(events)
	if err != nil {
		log.Printf("Failed to open terminal: %#v", err)
		return
	}

	for !m.Quit {
		handler := <-events
		if handler != nil {
			handler(&m)
		}
		select {
		case handler = <-events:
			if handler != nil {
				handler(&m)
			}
		default:
		}
		screen := view.Draw(&m)
		screen.Render(d, device.Position{X: 0, Y: 0}, device.Size(m.ScreenSize))
		d.Show()
	}

	lc.Stop()
	d.Stop()
}

// func startMock(path string, events model.EventHandler) {
// 	fsys := mock_fs.NewFs(path, events)
// 	fsys.Scan()
// }

// func startMock2(path string, events model.EventHandler) {
// 	fsys := mock2_fs.NewFs(path, events)
// 	fsys.Scan()
// }
