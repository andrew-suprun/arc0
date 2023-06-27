package model

import (
	"fmt"
	"path/filepath"
	"time"
)

type FileId struct {
	Root string
	Name string
}

type FileMeta struct {
	FileId
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
	Status ResulutionStatus
}

func (f *File) String() string {
	return fmt.Sprintf("File{Root: %q, Name: %q, Kind: %s, Status: %q, Hash: %q}", f.Root, f.Name, f.Kind, f.Status, f.Hash)
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

type ResulutionStatus int

const (
	Resolved ResulutionStatus = iota
	AutoResolve
	ResolveDuplicate
	ResolveAbsent
	Duplicate
	Absent
)

func (s ResulutionStatus) String() string {
	switch s {
	case Resolved:
		return ""
	case AutoResolve, ResolveDuplicate, ResolveAbsent:
		return "Pending"
	case Duplicate:
		return "Duplicate"
	case Absent:
		return "Absent"
	}
	return "UNKNOWN FILE STATUS"
}

func (s ResulutionStatus) Merge(other ResulutionStatus) ResulutionStatus {
	if s > other {
		return s
	}
	return other
}
