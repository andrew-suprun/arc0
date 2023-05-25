package main

import (
	"arch/device"
	"arch/device/tcell"
	"arch/files/file_fs"
	"arch/files/mock2_fs"
	"arch/files/mock_fs"
	"arch/lifecycle"
	"arch/model"
	"arch/view"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)

	lc := lifecycle.Lifecycle{}
	m := model.Model{}
	events := make(model.EventChan)

	if len(os.Args) >= 1 && os.Args[1] == "-sim" {
		fsys := mock_fs.NewFs(events)
		fsys.Scan("origin")
		fsys.Scan("copy 1")
		fsys.Scan("copy 2")
	} else if len(os.Args) >= 1 && os.Args[1] == "-sim2" {
		fsys := mock2_fs.NewFs(events)
		fsys.Scan("origin")
		fsys.Scan("copy 1")
		fsys.Scan("copy 2")
	} else {
		m.ArchivePaths = os.Args[1:]
		fsys := file_fs.NewFs(events, &lc)
		for _, path := range os.Args[1:] {
			err := fsys.Scan(path)
			if err != nil {
				log.Panicf("Failed to scan archive %s: %#v", path, err)
			}
		}
	}

	d, err := tcell.NewDevice(events)
	if err != nil {
		log.Printf("Failed to open terminal: %#v", err)
		return
	}

	logTotalFrames, logSkippedFrames := 0, 0
	for !m.Quit {
		handler := <-events
		log.Printf("###.1 handler %#v", handler)
		if handler != nil {
			handler.HandleEvent(&m)
		}
		select {
		case handler = <-events:
			log.Printf("###.2 handler %#v", handler)
			if handler != nil {
				handler.HandleEvent(&m)
				logSkippedFrames++
			}
		default:
		}
		screen := view.Draw(&m)
		screen.Render(d, device.Position{X: 0, Y: 0}, device.Size(m.ScreenSize))
		d.Show()
		logTotalFrames++
	}

	log.Println("### logTotalFrames", logTotalFrames)
	log.Println("### logSkippedFrames", logSkippedFrames)

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
