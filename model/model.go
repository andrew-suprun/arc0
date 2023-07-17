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

type File struct {
	Id
	Size    uint64
	ModTime time.Time
	Hash
}

func (m *File) String() string {
	return fmt.Sprintf("Meta{Root: %q, Path: %q Name: %q, Size: %d, ModTime: %s, Hash: %q}",
		m.Root, m.Path, m.Base, m.Size, m.ModTime.Format(time.DateTime), m.Hash)
}
