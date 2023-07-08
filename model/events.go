package model

import (
	"fmt"
)

type EventChan chan Event

type Event interface {
	event()
}

type ArchiveScanned struct {
	Root
	FileMetas
}

func (ArchiveScanned) event() {}

type FileHashed struct {
	FileId
	Hash
}

func (FileHashed) event() {}

func (h FileHashed) String() string {
	return fmt.Sprintf("Hashed Id: %q, Hash: %q", h.FileId.String(), h.Hash)
}

type FileDeleted DeleteFile

func (FileDeleted) event() {}

func (d FileDeleted) String() string {
	return fmt.Sprintf("Deleted Id: %q", FileId(d).String())
}

type FileRenamed RenameFile

func (FileRenamed) event() {}

func (r FileRenamed) String() string {
	return fmt.Sprintf("Renamed Id: %q, NewName: %q", r.FileId.String(), r.NewFullName.String())
}

type FileCopied CopyFile

func (FileCopied) event() {}

func (c FileCopied) String() string {
	return fmt.Sprintf("Copied From: %q, To: %q", c.From.String(), c.To.String())
}

type Progress struct {
	Root
	ProgressState
	HandledSize uint64
}

func (Progress) event() {}

type ProgressState int

const (
	Initial ProgressState = iota
	FileTreeScanned
	HashingFile
	FileTreeHashed
	CopyingFile
)

type Error struct {
	FullName
	Error error
}

func (Error) event() {}

type ScreenSize struct {
	Width, Height int
}

func (ScreenSize) event() {}

type Enter struct{}

func (Enter) event() {}

type Esc struct{}

func (Esc) event() {}

type RevealInFinder struct{}

func (RevealInFinder) event() {}

type SelectFirst struct{}

func (SelectFirst) event() {}

type SelectLast struct{}

func (SelectLast) event() {}

type MoveSelection struct{ Lines int }

func (MoveSelection) event() {}

type KeepOne struct{}

func (KeepOne) event() {}

type KeepAll struct{}

func (KeepAll) event() {}

type Tab struct{}

func (Tab) event() {}

type Delete struct{}

func (Delete) event() {}

type Scroll struct {
	Lines int
}

func (Scroll) event() {}

func (s Scroll) String() string {
	return fmt.Sprintf("Scroll(%#v)", s.Lines)
}

type MouseTarget struct{ Command any }

func (MouseTarget) event() {}

func (t MouseTarget) String() string {
	return fmt.Sprintf("MouseTarget(%q)", t.Command)
}

type PgUp struct{}

func (PgUp) event() {}

type PgDn struct{}

func (PgDn) event() {}

type Quit struct{}

func (Quit) event() {}
