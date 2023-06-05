package events

import (
	"fmt"
	"path/filepath"
	"time"
)

type EventChan chan Event

type Event interface {
	event()
}

type FileMeta struct {
	ArchivePath string
	Ino         uint64
	Path        string
	Name        string
	Size        uint64
	ModTime     time.Time
}

func (FileMeta) event() {}

func (m *FileMeta) String() string {
	return fmt.Sprintf("Meta{ArchivePath: %q, Path: %q, Name: %q, Size: %d, ModTime: %s}",
		m.ArchivePath, m.Path, m.Name, m.Size, m.ModTime.Format(time.DateTime))
}

func (f FileMeta) AbsName() string {
	return filepath.Join(f.ArchivePath, f.Path, f.Name)
}

type FileHash struct {
	ArchivePath string
	Ino         uint64
	Hash        string
}

func (FileHash) event() {}

type ScanProgress struct {
	ArchivePath  string
	ScanState    ScanState
	ScanProgress float64
}

func (ScanProgress) event() {}

type ScanState int

const (
	WalkFileTreeComplete ScanState = iota
	HashFileTree
	HashFileTreeComplete
)

type ScanError struct {
	Meta  FileMeta
	Error error
}

func (ScanError) event() {}

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
	return fmt.Sprintf("MouseTarget(%s)", t.Command)
}

type PgUp struct{}

func (PgUp) event() {}

type PgDn struct{}

func (PgDn) event() {}

type Quit struct{}

func (Quit) event() {}
