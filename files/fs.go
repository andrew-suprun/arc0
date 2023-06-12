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
	From events.FileMeta
}

func (Copy) msg() {}

type Move struct {
	From events.FileMeta
	To   events.FileMeta
}

func (Move) msg() {}

type Delete struct {
	File events.FileMeta
}

func (Delete) msg() {}
