package msg

import (
	"fmt"
	"strings"
	"time"
)

type CmdScan struct {
	Base string
}

type CmdQuit struct{}

type FileMeta struct {
	Ino     uint64
	Base    string
	Path    string
	Size    int
	ModTime time.Time
	Hash    string
}

type ScanError struct {
	Base  string
	Path  string
	Error error
}

type ScanStat struct {
	Base        string
	Path        string
	Size        int
	Hashed      int
	TotalSize   int
	TotalToHash int
	TotalHashed int
}

type FileMetas []*FileMeta

type ScanMetas struct {
	Base  string
	Metas FileMetas
}

// keys:          hash       base
type Analysis map[string]map[string]FileMetas

func (a Analysis) String() string {
	builder := &strings.Builder{}
	for hash, byBase := range a {
		fmt.Fprintln(builder, hash)
		for base, metas := range byBase {
			if len(metas) == 0 {
				continue
			}
			fmt.Fprintf(builder, "    %s\n", base)
			for _, meta := range metas {
				fmt.Fprintf(builder, "        %s\n", meta.Path)
			}
		}
	}
	return builder.String()
}

type ScanDone struct {
	Base string
}

type QuitApp struct{}
