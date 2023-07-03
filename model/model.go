package model

import (
	"fmt"
	"path/filepath"
	"time"
)

type Root string

func (root Root) String() string {
	return string(root)
}

type Path string

func (path Path) String() string {
	return string(path)
}

type Name string

func (name Name) String() string {
	return string(name)
}

type FullName struct {
	Path
	Name
}

func (name FullName) String() string {
	return filepath.Join(name.Path.String(), name.Name.String())
}

type FileId struct {
	Root
	Path
	Name
}

func (id FileId) AbsName() string {
	return filepath.Join(id.Root.String(), id.Path.String(), id.Name.String())
}

func (id FileId) FullName() FullName {
	return FullName{Path: id.Path, Name: id.Name}
}

type Hash string

func (hash Hash) String() string {
	return string(hash)
}

type FileMeta struct {
	FileId
	Size    uint64
	ModTime time.Time
}

func (m *FileMeta) String() string {
	return fmt.Sprintf("Meta{Root: %q, Path: %q Name: %q, Size: %d, ModTime: %s}",
		m.Root, m.Path, m.Name, m.Size, m.ModTime.Format(time.DateTime))
}

type File struct {
	FileMeta
	FileKind
	Hash
	Status
}

func (f *File) String() string {
	return fmt.Sprintf("File{Root: %q, Path: %q, Name: %q, Kind: %s, Size: %d, Status: %q, Hash: %q}", f.Root, f.Path, f.Name, f.FileKind, f.Size, f.Status, f.Hash)
}

func (f *File) StatusString() string {
	switch f.Status {
	case Resolved:
		return ""
	case Pending:
		return " Pending"
	case Duplicate:
		return " Duplicate"
	case Absent:
		return " Absent"
	}
	return "UNKNOWN FILE STATUS"
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
	Resolved Status = iota
	Pending
	Duplicate
	Absent
)

func (s Status) String() string {
	switch s {
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

func (f *File) MergeStatus(other *File) {
	if f.Status < other.Status {
		f.Status = other.Status
	}
}
