package model

import (
	"fmt"
	"time"
)

type Event interface {
	event()
}

type TotalSize struct {
	Root Root
	Size uint64
}

func (TotalSize) event() {}

type FileScanned struct {
	*File
}

func (FileScanned) event() {}

func (f *FileScanned) String() string {
	return f.File.String()
}

type ArchiveScanned struct {
	Root
}

func (ArchiveScanned) event() {}

type FileDeleted DeleteFile

func (FileDeleted) event() {}

func (h FileDeleted) String() string {
	return DeleteFile(h).String()
}

type FileRenamed RenameFile

func (FileRenamed) event() {}

func (h FileRenamed) String() string {
	return RenameFile(h).String()
}

type FileCopied CopyFile

func (FileCopied) event() {}

func (h FileCopied) String() string {
	return CopyFile(h).String()
}

type ProgressState int

const (
	Initial ProgressState = iota
	Scanned
)

type HashingProgress struct {
	Root
	Hashed uint64
}

func (HashingProgress) event() {}

type CopyingProgress uint64

func (CopyingProgress) event() {}

type Tick time.Time

func (Tick) event() {}

type Error struct {
	Id    Id
	Error error
}

func (Error) event() {}

type ScreenSize struct {
	Width, Height int
}

func (ScreenSize) event() {}

type Open struct{}

func (Open) event() {}

type Enter struct{}

func (Enter) event() {}

type Exit struct{}

func (Exit) event() {}

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
	Command any
	Lines   int
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

type Debug struct{}

func (Debug) event() {}

type Quit struct{}

func (Quit) event() {}
