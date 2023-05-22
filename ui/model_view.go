package ui

import (
	"arch/device"
	"arch/files"

	"os/exec"
	"path/filepath"
)

func Run(dev device.Device, fs files.FS, paths []string) {
	m := &model{
		paths:         paths,
		scanStates:    make([]*files.ScanState, len(paths)),
		scanResults:   make([]*files.ArchiveInfo, len(paths)),
		ctx:           &Context{Device: dev, Style: defaultStyle},
		sortAscending: []bool{true, true, true, true},
	}

	fsEvents := make(chan files.Event)
	for _, archive := range paths {
		go func(archive string) {
			for ev := range fs.Scan(archive) {
				fsEvents <- ev
			}
		}(archive)
	}
	deviceEvents := make(chan device.Event)
	go func() {
		for {
			deviceEvents <- dev.PollEvent()
		}
	}()

	running := true
	for running {
		select {
		case fsEvent := <-fsEvents:
			m.handleFilesEvent(fsEvent)
		case deviceEvent := <-deviceEvents:
			running = m.handleDeviceEvent(deviceEvent)
		}
		m.ctx.Reset()
		Column(0,
			m.title(),
			m.scanStats(),
			m.treeView(),
			m.statusLine(),
		).Render(m.ctx, Position{0, 0}, m.screenSize)
		m.ctx.Device.Render()
	}

	fs.Stop()
	dev.Stop()
}

func (m *model) enter() {
	selected := m.currentLocation().selected
	if selected == nil {
		return
	}
	if selected.kind == folder {
		m.locations = append(m.locations, location{file: selected})
		m.sort()
	} else {
		fileName := filepath.Join(selected.archive, selected.path, selected.name)
		exec.Command("open", fileName).Start()
	}
}

func (m *model) esc() {
	if len(m.locations) > 1 {
		m.locations = m.locations[:len(m.locations)-1]
		m.sort()
	}
}

func (m *model) up() {
	loc := m.currentLocation()
	if loc.selected != nil {
		for i, file := range loc.file.files {
			if file == loc.selected && i > 0 {
				loc.selected = loc.file.files[i-1]
				break
			}
		}
	} else {
		loc.selected = loc.file.files[len(loc.file.files)-1]
	}
}

func (m *model) down() {
	loc := m.currentLocation()
	if loc.selected != nil {
		for i, file := range loc.file.files {
			if file == loc.selected && i+1 < len(loc.file.files) {
				loc.selected = loc.file.files[i+1]
				break
			}
		}
	} else {
		loc.selected = loc.file.files[0]
	}
}

func (m *model) currentLocation() *location {
	if len(m.locations) == 0 {
		return nil
	}
	return &m.locations[len(m.locations)-1]
}
