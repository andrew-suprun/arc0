package controller

import (
	"arch/model"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

func (m *controller) handleEvent(event any) {
	if event == nil {
		return
	}

	switch event := event.(type) {
	case model.FileScanned:
		m.fileScanned(event)

	case model.FileHashed:
		m.fileHashed(event)

	case model.FileCopied:
		m.fileCopied(event)

	case model.FileRenamed:
		m.fileRenamed(event)

	case model.FileDeleted:
		m.fileDeleted(event)

	case model.Progress:
		m.progressEvent(event)

	case model.ScreenSize:
		m.screenSize = model.ScreenSize{Width: event.Width, Height: event.Height}

	case model.Enter:
		m.enter()

	case model.Esc:
		if m.currentPath == "" {
			return
		}
		parts := strings.Split(m.currentPath, "/")
		if len(parts) == 1 {
			m.currentPath = ""
		}
		m.currentPath = filepath.Join(parts[:len(parts)-1]...)

	case model.RevealInFinder:
		folder := m.folders[m.currentPath]
		if folder.selected != nil {
			exec.Command("open", "-R", folder.selected.AbsName()).Start()
		}

	case model.RenameSelection:
		m.moveSelection(event.Lines)
		m.makeSelectedVisible()

	case model.SelectFirst:
		m.selectFirst()
		m.makeSelectedVisible()

	case model.SelectLast:
		m.selectLast()
		m.makeSelectedVisible()

	case model.Scroll:
		m.shiftOffset(event.Lines)

	case model.MouseTarget:
		m.mouseTarget(event.Command)

	case model.PgUp:
		m.shiftOffset(-m.fileTreeLines)
		m.moveSelection(-m.fileTreeLines)

	case model.PgDn:
		m.shiftOffset(m.fileTreeLines)
		m.moveSelection(m.fileTreeLines)

	case model.KeepOne:
		m.keepSelected()

	case model.KeepAll:
		// TODO: Implement, maybe?

	case model.Delete:
		m.deleteSelected()

	case model.Error:
		m.Errors = append(m.Errors, event)

	case model.Quit:
		m.quit = true

	default:
		log.Panicf("### unhandled event: %#v", event)
	}
}
