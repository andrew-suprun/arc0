package model

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

type CopyFile FileMeta

func (CopyFile) msg() {}

type RenameFile struct {
	OldMeta FileMeta
	NewMeta FileMeta
}

func (RenameFile) msg() {}

type DeleteFile FileMeta

func (DeleteFile) msg() {}
