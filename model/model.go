package model

import (
	"fmt"
	"path/filepath"
	"time"
)

type FileMeta struct {
	Root    string
	Name    string
	Size    uint64
	ModTime time.Time
}

func (m *FileMeta) String() string {
	return fmt.Sprintf("Meta{Root: %q, Name: %q, Size: %d, ModTime: %s}",
		m.Root, m.Name, m.Size, m.ModTime.Format(time.DateTime))
}

func (f FileMeta) AbsName() string {
	return filepath.Join(f.Root, f.Name)
}

type File struct {
	FileMeta
	Kind   FileKind
	Hash   string
	Status Status
}

func (f *File) String() string {
	return fmt.Sprintf("File{Meta: %v, Kind: %s, Status: %q, Hash: %q}", f.FileMeta.String(), f.Kind, f.Status, f.Hash)
}

type Files []*File

type FileKind int

const (
	FileRegular FileKind = iota
	FileFolder
)

func (k FileKind) String() string {
	switch k {
	case FileFolder:
		return "FileFolder"
	case FileRegular:
		return "FileRegular"
	}
	return "UNKNOWN FILE KIND"
}

type Status int

const (
	Identical Status = iota
	Resolved
	Pending
	Duplicate
	Absent
)

func (s Status) String() string {
	switch s {
	case Identical:
		return ""
	case Resolved:
		return "Resolved"
	case Pending:
		return "Pending"
	case Duplicate:
		return "Duplicate"
	case Absent:
		return "Absent"
	}
	return "UNKNOWN FILE STATUS"
}

func (s Status) Merge(other Status) Status {
	if s > other {
		return s
	}
	return other
}
