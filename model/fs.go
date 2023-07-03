package model

import (
	"arch/actor"
	"fmt"
	"strings"
)

type FS interface {
	NewArchiveScanner(root Root) ArchiveScanner
	actor.Actor[HandleFiles]
}

type ArchiveScanner interface {
	ScanArchive()
	HashArchive()
}

type HandleFiles struct {
	Hash
	Delete []DeleteFile
	Rename []RenameFile
	Copy   *CopyFile
}

func (h HandleFiles) String() string {
	buf := &strings.Builder{}
	fmt.Fprintf(buf, "hash: %q\n", h.Hash)
	for _, d := range h.Delete {
		fmt.Fprintf(buf, "    delete: %q/%q/%q\n", d.Root, d.Path, d.Name)
	}

	for _, r := range h.Rename {
		fmt.Fprintf(buf, "    rename: root %q: %q/%q -> %q/%q\n", r.Root, r.Path, r.Name, r.NewPath, r.NewName)
	}
	if h.Copy != nil {
		fmt.Fprintf(buf, "    copy: %q/%q/%q\n", h.Copy.Root, h.Copy.Path, h.Copy.Name)
		for _, t := range h.Copy.TargetRoots {
			fmt.Fprintf(buf, "          -> %q\n", t)
		}
	}
	return buf.String()
}

type DeleteFile FileId

type RenameFile struct {
	FileId
	NewPath Path
	NewName Name
}

type CopyFile struct {
	FileId
	TargetRoots []Root
}
