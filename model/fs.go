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
	Root  string
	INode uint64
}

func (CopyFile) msg() {}

type RenameFile struct {
	INode   uint64
	NewName string
}

func (RenameFile) msg() {}

type DeleteFile struct {
	INode uint64
}

func (DeleteFile) msg() {}
