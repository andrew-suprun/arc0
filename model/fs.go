package model

import "arch/actor"

type FS interface {
	NewArchiveScanner(root Root) ArchiveScanner
}

type ArchiveScanner interface {
	actor.Actor[FileCommand]
}

type FileCommand interface {
	cmd()
}

type ScanArchive struct{}

func (ScanArchive) cmd() {}

type HashArchive struct{}

func (HashArchive) cmd() {}

type DeleteFile FileId

func (DeleteFile) cmd() {}

type RenameFile struct {
	FileId
	NewFullName FullName
}

func (RenameFile) cmd() {}

type CopyFile struct {
	From FileId
	To   Root
}

func (CopyFile) cmd() {}
