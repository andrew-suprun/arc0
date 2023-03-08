package msg

import (
	"fmt"
	"strings"
	"time"
)

type CmdScan struct {
	Base  string
	Index int
}

type CmdQuit struct{}

type FileInfo struct {
	Ino     uint64
	Base    string
	Path    string
	Size    int
	ModTime time.Time
	Hash    string
}

type ArchiveInfo []FileInfo

func (a ArchiveInfo) String() string {
	hash := ""
	base := ""
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "ArchiveInfo [%d]\n", len(a))
	for _, info := range a {
		if hash != info.Hash {
			hash = info.Hash
			fmt.Fprintln(builder, hash)
			base = ""
		}
		if base != info.Base {
			base = info.Base
			fmt.Fprintf(builder, "    %s\n", base)
		}
		fmt.Fprintf(builder, "        %s\n", info.Path)
	}
	return builder.String()
}

type ScanError struct {
	Base  string
	Path  string
	Error error
}

type ScanState struct {
	Base        string
	Path        string
	Size        int
	Hashed      int
	TotalSize   int
	TotalToHash int
	TotalHashed int
}

type ScanDone struct {
	Base string
}

type QuitApp struct{}
