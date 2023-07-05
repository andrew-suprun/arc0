package controller

import (
	m "arch/model"
	"log"
)

func (c *controller) handleEvent(event any) {
	if event == nil {
		return
	}

	switch event := event.(type) {
	case m.ArchiveScanned:
		c.archiveScanned(event)
		c.buildEntries()

	case m.FileHashed:
		c.fileHashed(event)
		c.buildEntries()

	case m.FileDeleted:
		c.fileDeleted(event)
		c.buildEntries()

	case m.FileRenamed:
		c.fileRenamed(event)
		c.buildEntries()

	case m.FileCopied:
		c.fileCopied(event)
		c.buildEntries()

	case m.ScanProgress:
		c.scanProgress(event)

	case m.FileCopyProgress:
		c.fileCopyProgress(event)

	case m.ScreenSize:
		c.screenSize = m.ScreenSize{Width: event.Width, Height: event.Height}

	case m.Enter:
		c.enter()
		c.buildEntries()

	case m.Esc:
		c.esc()
		c.buildEntries()

	case m.RevealInFinder:
		c.revealInFinder()

	case m.MoveSelection:
		c.moveSelection(event.Lines)

	case m.SelectFirst:
		c.selectFirst()

	case m.SelectLast:
		c.selectLast()

	case m.Scroll:
		c.shiftOffset(event.Lines)

	case m.MouseTarget:
		c.mouseTarget(event.Command)
		c.buildEntries()

	case m.PgUp:
		c.pgUp()

	case m.PgDn:
		c.pgDn()

	case m.Tab:
		c.tab()
		c.buildEntries()

	case m.KeepOne:
		c.keepSelected()
		c.buildEntries()

	case m.KeepAll:
		// TODO: Implement, maybe?

	case m.Delete:
		c.deleteSelected()
		c.buildEntries()

	case m.Error:
		c.Errors = append(c.Errors, event)

	case m.Quit:
		c.quit = true

	default:
		log.Panicf("### unhandled event: %#v", event)
	}
}
