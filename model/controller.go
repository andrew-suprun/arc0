package model

import (
	"arch/events"
	"log"
	"os/exec"
	"path/filepath"
	"time"
)

func (m *model) handleEvent(event any) {
	if event == nil {
		return
	}

	switch event := event.(type) {
	case events.FileMeta:
		m.fileMeta(event)

	case events.FileHash:
		m.fileHash(event)

	case events.ScanProgress:
		m.scanProgressEvent(event)

	case events.ScanError:
		m.archives[m.archiveIdx(event.Meta.ArchivePath)].totalHashed += event.Meta.Size
		m.Errors = append(m.Errors, event)

	case events.ScreenSize:
		m.screenSize = ScreenSize{Width: event.Width, Height: event.Height}

	case events.Enter:
		m.enter()

	case events.Esc:
		if len(m.breadcrumbs) > 1 {
			m.breadcrumbs = m.breadcrumbs[:len(m.breadcrumbs)-1]
			m.sort()
		}

	case events.RevealInFinder:
		selected := m.currentFolder().selected
		if selected != nil {
			exec.Command("open", "-R", selected.AbsName()).Start()
		}

	case events.MoveSelection:
		m.moveSelection(event.Lines)
		m.makeSelectedVisible()

	case events.SelectFirst:
		m.selectFirst()
		m.makeSelectedVisible()

	case events.SelectLast:
		m.selectLast()
		m.makeSelectedVisible()

	case events.Scroll:
		m.shiftOffset(event.Lines)

	case events.MouseTarget:
		m.mouseTarget(event.Command)

	case events.PgUp:
		m.shiftOffset(-m.fileTreeLines)
		m.moveSelection(-m.fileTreeLines)

	case events.PgDn:
		m.shiftOffset(m.fileTreeLines)
		m.moveSelection(m.fileTreeLines)

	case events.Quit:
		m.quit = true

	default:
		log.Panicf("### unhandled event: %#v", event)
	}
}

func (m *model) mouseTarget(cmd any) {
	folder := m.currentFolder()
	switch cmd := cmd.(type) {
	case selectFile:
		if folder.selected == cmd && time.Since(m.lastMouseEventTime).Seconds() < 0.5 {
			m.enter()
		} else {
			folder.selected = cmd
		}
		m.lastMouseEventTime = time.Now()

	case sortColumn:
		if cmd == folder.sortColumn {
			folder.sortAscending[folder.sortColumn] = !folder.sortAscending[folder.sortColumn]
		} else {
			folder.sortColumn = cmd
		}

		m.sort()
	}

}

func (m *model) selectFirst() {
	m.currentFolder().selected = m.currentFolder().entries[0]
}

func (m *model) selectLast() {
	folder := m.currentFolder()
	entries := folder.entries
	folder.selected = entries[len(entries)-1]
}

func (m *model) moveSelection(lines int) {
	folder := m.currentFolder()
	selected := folder.selected
	if selected == nil {
		if lines > 0 {
			m.selectFirst()
		} else if lines < 0 {
			m.selectLast()
		}
	}
	entries := folder.entries
	idxSelected := 0
	foundSelected := false

	for i := 0; i < len(entries); i++ {
		if entries[i] == selected {
			idxSelected = i
			foundSelected = true
			break
		}
	}
	if foundSelected {
		idxSelected += lines
		if idxSelected < 0 {
			idxSelected = 0
		} else if idxSelected >= len(entries) {
			idxSelected = len(entries) - 1
		}
		folder.selected = entries[idxSelected]
	}
}

func (m *model) enter() {
	selected := m.currentFolder().selected
	if selected != nil {
		if selected.Kind == FileFolder {
			path := filepath.Join(selected.Path, selected.Name)
			m.breadcrumbs = append(m.breadcrumbs, m.folders[path])
			m.sort()
		} else {
			exec.Command("open", selected.AbsName()).Start()
		}
	}
}

func (m *model) archiveIdx(archivePath string) int {
	for i, archive := range m.archives {
		if archivePath == archive.archivePath {
			return i
		}
	}
	log.Panicf("### Invalid archive path: %q", archivePath)
	return -1
}

func (m *model) shiftOffset(lines int) {
	folder := m.currentFolder()
	nEntries := len(folder.entries)
	folder.lineOffset += lines
	if folder.lineOffset < 0 {
		folder.lineOffset = 0
	} else if folder.lineOffset >= nEntries {
		folder.lineOffset = nEntries - 1
	}
}
