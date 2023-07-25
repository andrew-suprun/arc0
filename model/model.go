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

func (p Path) ParentName() Name {
	strPath := p.String()
	path := Path(filepath.Dir(strPath))
	if path == "." {
		path = ""
	}
	return Name{Path: path, Base: Base(filepath.Base(strPath))}
}

type Base string

func (name Base) String() string {
	return string(name)
}

type Name struct {
	Path
	Base
}

func (name Name) String() string {
	return filepath.Join(name.Path.String(), name.Base.String())
}

type Id struct {
	Root
	Name
}

func (id Id) String() string {
	return filepath.Join(id.Root.String(), id.Path.String(), id.Base.String())
}

type Hash string

func (hash Hash) String() string {
	return string(hash)
}

type Meta struct {
	Id
	Size    uint64
	ModTime time.Time
	Hash
}

func (m *Meta) String() string {
	return fmt.Sprintf("Meta{Root: %q, Path: %q Name: %q, Size: %d, ModTime: %s, Hash: %q}",
		m.Root, m.Path, m.Base, m.Size, m.ModTime.Format(time.DateTime), m.Hash)
}

type File struct {
	Meta
	Kind
	State
}

func (f *File) String() string {
	return fmt.Sprintf("File{FileId: %q, Kind: %s, Size: %d, Hash: %q, State: %s}", f.Id, f.Kind, f.Size, f.Hash, f.State)
}

type Kind int

const (
	FileRegular Kind = iota
	FileFolder
)

func (k Kind) String() string {
	switch k {
	case FileFolder:
		return "FileFolder"
	case FileRegular:
		return "FileRegular"
	}
	return "UNKNOWN FILE KIND"
}

type State int

const (
	Initial State = iota
	Resolved
	Pending
	Autoresolve
	Duplicate
	Absent
)

func (s State) String() string {
	switch s {
	case Initial:
		return "Initial"
	case Resolved:
		return "Resolved"
	case Pending:
		return "Pending"
	case Autoresolve:
		return "Autoresolve"
	case Duplicate:
		return "Duplicate"
	case Absent:
		return "Absent"
	}
	return "UNKNOWN FILE STATE"
}

type SortColumn int

const (
	SortByName SortColumn = iota
	SortByTime
	SortBySize
)

func (c SortColumn) String() string {
	switch c {
	case SortByName:
		return "SortByName"
	case SortByTime:
		return "SortByTime"
	case SortBySize:
		return "SortBySize"
	}
	return "Illegal Sort Solumn"
}
