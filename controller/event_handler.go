package controller

import (
	m "arch/model"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

func (c *controller) handleEvent(event any) {
	if event == nil {
		return
	}

	switch event := event.(type) {
	case m.ArchiveScanned:
		c.archiveScanned(event)

	case m.FileHashed:
		c.fileHashed(event)

	case m.FilesHandled:
		c.filesHandled(event)

	case m.ScanProgress:
		c.scanProgress(event)

	case m.FileCopyProgress:
		c.fileCopyProgress(event)

	case m.ScreenSize:
		c.screenSize = m.ScreenSize{Width: event.Width, Height: event.Height}

	case m.Enter:
		c.enter()

	case m.Esc:
		if c.currentPath == "" {
			return
		}
		parts := strings.Split(c.currentPath.String(), "/")
		if len(parts) == 1 {
			c.currentPath = ""
		}
		c.currentPath = m.Path(filepath.Join(parts[:len(parts)-1]...))

	case m.RevealInFinder:
		folder := c.folders[c.currentPath]
		if folder.selected != nil {
			exec.Command("open", "-R", folder.selected.AbsName()).Start()
		}

	case m.MoveSelection:
		c.moveSelection(event.Lines)
		c.makeSelectedVisible()

	case m.SelectFirst:
		c.selectFirst()
		c.makeSelectedVisible()

	case m.SelectLast:
		c.selectLast()
		c.makeSelectedVisible()

	case m.Scroll:
		c.shiftOffset(event.Lines)

	case m.MouseTarget:
		c.mouseTarget(event.Command)

	case m.PgUp:
		c.shiftOffset(-c.fileTreeLines)
		c.moveSelection(-c.fileTreeLines)

	case m.PgDn:
		c.shiftOffset(c.fileTreeLines)
		c.moveSelection(c.fileTreeLines)

	case m.KeepOne:
		c.keepSelected()

	case m.Tab:
		c.tab()

	case m.KeepAll:
		// TODO: Implement, maybe?

	case m.Delete:
		c.deleteSelected()

	case m.Error:
		c.Errors = append(c.Errors, event)

	case m.Quit:
		c.quit = true

	default:
		log.Panicf("### unhandled event: %#v", event)
	}
}
