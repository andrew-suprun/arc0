package model

import (
	"fmt"
	"path/filepath"
	"time"
)

type FileMeta struct {
	Root    string
	INode   uint64
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
	Status FileStatus
	Hash   string
	Counts []int
}

func (f File) String() string {
	return fmt.Sprintf("File{Meta: %s, Kind: %s, Status: %s}", f.FileMeta.String(), f.Kind, f.Status)
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

type FileStatus int

const (
	Identical FileStatus = iota
	Pending
	Resolved
	Conflict
)

func (s FileStatus) String() string {
	switch s {
	case Identical:
		return "Identical"
	case Pending:
		return "Pending"
	case Resolved:
		return "Resolved"
	case Conflict:
		return "Conflict"
	}
	return "UNKNOWN FILE STATUS"
}

func (s FileStatus) Merge(other FileStatus) FileStatus {
	if s > other {
		return s
	}
	return other
}
