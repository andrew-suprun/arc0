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

func (id FileId) String() string {
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

type FileMetas []FileMeta
