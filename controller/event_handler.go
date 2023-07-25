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

	case m.FileHashed:
		c.fileHashed(event)

	case m.FileDeleted:
		c.fileDeleted(event)

	case m.FileRenamed:
		c.fileRenamed(event)

	case m.FileCopied:
		c.fileCopied(event)

	case m.HashingProgress:
		c.handleHashingProgress(event)

	case m.CopyingProgress:
		c.handleCopyingProgress(event)

	case m.Tick:
		c.handleTick(event)

	case m.ScreenSize:
		c.screenSize = m.ScreenSize{Width: event.Width, Height: event.Height}

	case m.Enter:
		c.enter()

	case m.Open:
		c.open()

	case m.Exit:
		c.exit()

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

	case m.PgUp:
		c.pgUp()

	case m.PgDn:
		c.pgDn()

	case m.Tab:
		c.tab()

	case m.KeepOne:
		c.keepSelected()

	case m.KeepAll:
		// TODO: Implement, maybe?

	case m.Delete:
		folder := c.currentFolder()
		c.deleteFile(folder.selectedEntry)

	case m.Error:
		log.Printf("### Error: %s", event)
		c.Errors = append(c.Errors, event)

	case m.Quit:
		c.quit = true

	case m.Debug:
		log.Println(c.screenString())

	default:
		log.Panicf("### unhandled event: %#v", event)
	}
}
