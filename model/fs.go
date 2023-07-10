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

type HashArchive struct{}

func (HashArchive) cmd() {}

type HandleFiles struct {
	Hash
	Delete []Id
	Rename []RenameFile
	Copy   *CopyFile
}

func (HandleFiles) cmd() {}

func (h HandleFiles) String() string {
	buf := &strings.Builder{}
	fmt.Fprintf(buf, "hash: %q\n", h.Hash)
	for _, d := range h.Delete {
		fmt.Fprintf(buf, "    delete: %q/%q\n", d.Root, d.Base)
	}

	for _, r := range h.Rename {
		fmt.Fprintf(buf, "    rename: id %q: new name %q\n", r.Id, r.NewFullName)
	}
	if h.Copy != nil {
		fmt.Fprintf(buf, "    copy: from %q\n", h.Copy.From)
		for _, t := range h.Copy.To {
			fmt.Fprintf(buf, "          -> %q\n", t)
		}
	}
	return buf.String()
}

type RenameFile struct {
	Id
	NewFullName Name
}

type CopyFile struct {
	From Id
	To   []Root
}
