package model

import (
	"fmt"
)

type EventChan chan Event

type Event interface {
	event()
}

type ArchiveScanned struct {
	Root  string
	Metas []FileMeta
}

func (ArchiveScanned) event() {}

type FileHashed struct {
	Root, Name, Hash string
}

func (FileHashed) event() {}

type FilesHandled HandleFiles

func (h FilesHandled) String() string {
	return HandleFiles(h).String()
}

func (FilesHandled) event() {}

type Progress struct {
	Root          string
	ProgressState ProgressState
	Processed     uint64
}

func (Progress) event() {}

type ProgressState int

const (
	Initial ProgressState = iota
	FileTreeScanned
	HashingFileTree
	FileTreeHashed
)

type Error struct {
	Meta  FileMeta
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

type RenameSelection struct{ Lines int }

func (RenameSelection) event() {}

type KeepOne struct{}

func (KeepOne) event() {}

type KeepAll struct{}

func (KeepAll) event() {}

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
