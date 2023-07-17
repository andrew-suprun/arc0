package model

import (
	"arch/actor"
	"fmt"
	"strings"
)

type FS interface {
	NewArchiveScanner(root Root) ArchiveScanner
}

type ArchiveScanner interface {
	actor.Actor[FileCommand]
}

type FileCommand interface {
	cmd()
}

type ScanArchive struct{}

func (ScanArchive) cmd() {}

type HandleFiles struct {
	Hash
	Delete []Id
	Rename []RenameFile
	Copy   *CopyFile
}

func (HandleFiles) cmd() {}

func (h HandleFiles) String() string {
	buf := &strings.Builder{}
	fmt.Fprintf(buf, "HandleFiles: hash: %q\n", h.Hash)
	for _, d := range h.Delete {
		fmt.Fprintf(buf, "    delete: %s\n", d)
	}

	for _, r := range h.Rename {
		fmt.Fprintf(buf, "    %s\n", r)
	}
	if h.Copy != nil {
		fmt.Fprintf(buf, "    %s\n", h.Copy)
	}
	return buf.String()
}

type RenameFile struct {
	Id
	NewId Id
}

func (r RenameFile) String() string {
	return fmt.Sprintf("RenameFile: Id=%q, NewId=%q", r.Id, r.NewId)
}

type CopyFile struct {
	From Id
	To   []Root
}

func (c CopyFile) String() string {
	return fmt.Sprintf("CopyFile: From=%q, To=%v", c.From, c.To)
}
