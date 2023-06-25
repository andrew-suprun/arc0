package model

import (
	"arch/actor"
	"fmt"
	"strings"
)

type FS interface {
	ScanArchive(root string)
	NewFileHandler() actor.Actor[HandleFiles]
}

type HandleFiles struct {
	Hash   string
	Delete []DeleteFile
	Rename []RenameFile
	Copy   *CopyFile
}

func (h HandleFiles) String() string {
	buf := &strings.Builder{}
	fmt.Fprintf(buf, "hash: %q\n", h.Hash)
	for _, d := range h.Delete {
		fmt.Fprintf(buf, "    delete: %q/%q\n", d.Root, d.Name)
	}

	for _, r := range h.Rename {
		fmt.Fprintf(buf, "    rename: root %q: %q -> %q\n", r.Root, r.OldName, r.NewName)
	}
	if h.Copy != nil {
		fmt.Fprintf(buf, "    copy: %q/%q\n", h.Copy.SourceRoot, h.Copy.Name)
		for _, t := range h.Copy.TargetRoots {
			fmt.Fprintf(buf, "          -> %q\n", t)
		}
	}
	return buf.String()
}

type DeleteFile struct {
	Root string
	Name string
}

type RenameFile struct {
	Root    string
	OldName string
	NewName string
}

type CopyFile struct {
	SourceRoot  string
	TargetRoots []string
	Name        string
}
