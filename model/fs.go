package model

type FS interface {
	NewScanner(root string) Scanner
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

type CopyFile struct {
	Root string
	Name string
}

func (CopyFile) msg() {}

type RenameFile struct {
	OldName string
	NewName string
}

func (RenameFile) msg() {}

type DeleteFile struct {
	Name string
}

func (DeleteFile) msg() {}
