package mock_fs

import (
	"arch/events"
	"arch/files"
	"log"
	"math/rand"
	"strings"
	"time"
)

type mockFs struct {
	scan   bool
	events events.EventChan
}

func NewFs(events events.EventChan, scan bool) files.FS {
	return &mockFs{
		scan:   scan,
		events: events,
	}
}

type scanner struct {
	scan        bool
	events      events.EventChan
	archivePath string
	totalSize   uint64
	totalHashed uint64
}

func (fsys *mockFs) NewScanner(archivePath string) files.Scanner {
	return &scanner{
		scan:        fsys.scan,
		events:      fsys.events,
		archivePath: archivePath,
	}
}

func (s *scanner) Handler(msg files.Msg) bool {
	switch msg.(type) {
	case files.ScanArchive:
		return s.scanArchive()
	case files.HashArchive:
		return s.hashArchive()
	}
	log.Panicf("### ERROR: Unhandled scanner message: %#v", msg)
	return false
}

func (s *scanner) scanArchive() bool {
	archFiles := metas[s.archivePath]
	go func() {
		for _, meta := range archFiles {
			s.events <- events.FileMeta{
				INode:       meta.INode,
				ArchivePath: s.archivePath,
				FullName:    meta.FullName,
				Size:        meta.Size,
				ModTime:     meta.ModTime,
			}
			s.totalSize += meta.Size
		}
		s.events <- events.ScanProgress{
			ArchivePath: s.archivePath,
			ScanState:   events.WalkFileTreeComplete,
		}
	}()
	return true
}

func (s *scanner) hashArchive() bool {
	archFiles := metas[s.archivePath]
	scans := make([]bool, len(archFiles))

	for i := range archFiles {
		scans[i] = s.scan && rand.Intn(2) == 0
	}
	go func() {
		for i := range archFiles {
			if !scans[i] {
				meta := archFiles[i]
				s.events <- events.FileHash{
					INode:       meta.INode,
					ArchivePath: meta.ArchivePath,
					Hash:        meta.Hash,
				}
				s.totalHashed += meta.Size
				s.events <- events.ScanProgress{
					ArchivePath:  s.archivePath,
					ScanState:    events.HashFileTree,
					ScanProgress: float64(s.totalHashed) / float64(s.totalSize),
				}
			}
		}
		for i := range archFiles {
			if scans[i] {
				meta := archFiles[i]
				for hashed := uint64(0); ; hashed += 20000 {
					if hashed > meta.Size {
						hashed = meta.Size
					}
					s.events <- events.ScanProgress{
						ArchivePath:  meta.ArchivePath,
						ScanState:    events.HashFileTree,
						ScanProgress: float64(s.totalHashed+hashed) / float64(s.totalSize),
					}
					if hashed == meta.Size {
						break
					}
					time.Sleep(time.Millisecond)
				}
				s.totalHashed += meta.Size
				s.events <- events.FileHash{
					INode:       meta.INode,
					ArchivePath: meta.ArchivePath,
					Hash:        meta.Hash,
				}
			}
		}
		s.events <- events.ScanProgress{
			ArchivePath: s.archivePath,
			ScanState:   events.HashFileTreeComplete,
		}
	}()
	return true
}

var beginning = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
var end = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
var duration = end.Sub(beginning)

type fileMeta struct {
	INode       uint64
	ArchivePath string
	FullName    string
	Hash        string
	Size        uint64
	ModTime     time.Time
}

var sizes = map[string]uint64{}
var modTimes = map[string]time.Time{}
var inode = uint64(0)

func init() {
	sizes["yyyy"] = 50000000
	sizes["hhhh"] = 50000000
	for archPath, metaStrings := range metaMap {
		for _, meta := range metaStrings {
			parts := strings.Split(meta, ":")
			name := parts[0]
			hash := parts[1]
			size, ok := sizes[hash]
			if !ok {
				size = uint64(rand.Intn(100000000))
				sizes[hash] = size
			}
			modTime, ok := modTimes[hash]
			if !ok {
				modTime = beginning.Add(time.Duration(rand.Int63n(int64(duration))))
				modTimes[hash] = modTime
			}
			inode++
			file := &fileMeta{
				INode:       inode,
				ArchivePath: archPath,
				FullName:    name,
				Hash:        hash,
				Size:        size,
				ModTime:     modTime,
			}
			metas[archPath] = append(metas[archPath], file)
		}
	}
}

var metas = map[string][]*fileMeta{}
var metaMap = map[string][]string{
	"origin": {
		"a/b/c/x.txt:hhhh",
		"a/b/e/f.txt:gggg",
		"a/b/e/g.txt:tttt",
		"x.txt:hhhh",
		"q/w/e/r/t/y.txt:qwerty",
		"yyy.txt:yyyy",
	},
	"copy 1": {
		"a/b/c/d.txt:llll",
		"a/b/e/f.txt:hhhh",
		"a/b/e/g.txt:tttt",
		"x.txt:mmmm",
		"y.txt:gggg",
		"a/b/c/x.txt:hhhh",
		"zzzz.txt:hhhh",
		"x/y/z.txt:zzzz",
		"yyy.txt:yyyy",
	},
	"copy 2": {
		"a/b/c/f.txt:hhhh",
		"a/b/e/x.txt:gggg",
		"a/b/e/g.txt:tttt",
		"x:asdfg",
		"q/w/e/r/t/y.txt:12345",
		"yyy.txt:yyyy",
	},
}
