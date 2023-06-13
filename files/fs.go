package files

import "arch/events"

type FS interface {
	NewScanner(archivePath string) Scanner
}

type Scanner interface {
	Handler(msg Msg) bool
}

type Msg interface {
	msg()
}

type ScanArchive struct{}

func (ScanArchive) msg() {}

type HashArchive struct{}

func (HashArchive) msg() {}

type Copy struct {
	Source events.FileMeta
}

func (Copy) msg() {}

type Move struct {
	OldMeta events.FileMeta
	NewMeta events.FileMeta
}

func (Move) msg() {}

type Delete struct {
	File events.FileMeta
}

func (Delete) msg() {}
