package controller

import (
	"arch/model"
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
	case model.ArchiveScanned:
		c.archiveScanned(event)

	case model.FileHashed:
		c.fileHashed(event)

	case model.FilesHandled:
		c.filesHandled(event)

	case model.ScanProgress:
		c.scanProgress(event)

	case model.FileCopyProgress:
		c.fileCopyProgress(event)

	case model.ScreenSize:
		c.screenSize = model.ScreenSize{Width: event.Width, Height: event.Height}

	case model.Enter:
		c.enter()

	case model.Esc:
		if c.currentPath == "" {
			return
		}
		parts := strings.Split(c.currentPath, "/")
		if len(parts) == 1 {
			c.currentPath = ""
		}
		c.currentPath = filepath.Join(parts[:len(parts)-1]...)

	case model.RevealInFinder:
		folder := c.folders[c.currentPath]
		if folder.selected != nil {
			exec.Command("open", "-R", folder.selected.AbsName()).Start()
		}

	case model.RenameSelection:
		c.moveSelection(event.Lines)
		c.makeSelectedVisible()

	case model.SelectFirst:
		c.selectFirst()
		c.makeSelectedVisible()

	case model.SelectLast:
		c.selectLast()
		c.makeSelectedVisible()

	case model.Scroll:
		c.shiftOffset(event.Lines)

	case model.MouseTarget:
		c.mouseTarget(event.Command)

	case model.PgUp:
		c.shiftOffset(-c.fileTreeLines)
		c.moveSelection(-c.fileTreeLines)

	case model.PgDn:
		c.shiftOffset(c.fileTreeLines)
		c.moveSelection(c.fileTreeLines)

	case model.KeepOne:
		c.keepSelected()

	case model.Tab:
		c.tab()

	case model.KeepAll:
		// TODO: Implement, maybe?

	case model.Delete:
		c.deleteSelected()

	case model.Error:
		c.Errors = append(c.Errors, event)

	case model.Quit:
		c.quit = true

	default:
		log.Panicf("### unhandled event: %#v", event)
	}
}
