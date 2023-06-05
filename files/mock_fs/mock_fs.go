package mock_fs

import (
	"arch/events"
	"arch/files"
	"math/rand"
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

func (s *scanner) ScanArchive() {
	archFiles := metas[s.archivePath]
	scans := make([]bool, len(archFiles))

	for i := range archFiles {
		size := uint64(rand.Intn(100000000))
		s.totalSize += size
		archFiles[i].Ino = uint64(i)
		archFiles[i].Size = size
		archFiles[i].ModTime = beginning.Add(time.Duration(rand.Int63n(int64(duration))))
		scans[i] = s.scan && rand.Intn(2) == 0
	}
	go func() {
		for _, meta := range archFiles {
			s.events <- events.FileMeta{
				Ino:         meta.Ino,
				ArchivePath: s.archivePath,
				Path:        meta.Path,
				Name:        meta.Name,
				Size:        meta.Size,
				ModTime:     meta.ModTime,
			}
		}
		s.events <- events.ScanProgress{
			ArchivePath: s.archivePath,
			ScanState:   events.WalkFileTreeComplete,
		}
		for i := range archFiles {
			if !scans[i] {
				meta := archFiles[i]
				s.events <- events.FileHash{
					Ino:         meta.Ino,
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
					Ino:         meta.Ino,
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
}

func (s *scanner) HashArchive() {
}

var beginning = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
var end = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
var duration = end.Sub(beginning)

type fileMeta struct {
	Ino         uint64
	ArchivePath string
	Path        string
	Name        string
	Hash        string
	Size        uint64
	ModTime     time.Time
}

var metas = map[string][]*fileMeta{
	"origin": {
		{
			ArchivePath: "origin",
			Path:        "a/b/c",
			Name:        "x.txt",
			Hash:        "hhhh",
		},
		{
			ArchivePath: "origin",
			Path:        "a/b/e",
			Name:        "f.txt",
			Hash:        "gggg",
		},
		{
			ArchivePath: "origin",
			Path:        "a/b/e",
			Name:        "g.txt",
			Hash:        "tttt",
		},
		{
			ArchivePath: "origin",
			Path:        "",
			Name:        "x.txt",
			Hash:        "hhhh",
		},
		{
			ArchivePath: "origin",
			Path:        "q/w/e/r/t",
			Name:        "y.txt",
			Hash:        "qwerty",
		},
		{
			ArchivePath: "origin",
			Path:        "",
			Name:        "yyy.txt",
			Hash:        "yyyy",
		},
	},
	"copy 1": {
		{
			ArchivePath: "copy 1",
			Path:        "a/b/c",
			Name:        "d.txt",
			Hash:        "llll",
		},
		{
			ArchivePath: "copy 1",
			Path:        "a/b/e",
			Name:        "f.txt",
			Hash:        "hhhh",
		},
		{
			ArchivePath: "copy 1",
			Path:        "a/b/e",
			Name:        "g.txt",
			Hash:        "tttt",
		},
		{
			ArchivePath: "copy 1",
			Path:        "",
			Name:        "x.txt",
			Hash:        "mmmm",
		},
		{
			ArchivePath: "copy 1",
			Path:        "",
			Name:        "y.txt",
			Hash:        "gggg",
		},
		{
			ArchivePath: "copy 1",
			Path:        "a/b/c",
			Name:        "x.txt",
			Hash:        "hhhh",
		},
		{
			ArchivePath: "copy 1",
			Path:        "",
			Name:        "zzzz.txt",
			Hash:        "hhhh",
		},
		{
			ArchivePath: "copy 1",
			Path:        "x/y",
			Name:        "z.txt",
			Hash:        "zzzz",
		},
		{
			ArchivePath: "copy 1",
			Path:        "",
			Name:        "yyy.txt",
			Hash:        "yyyy",
		},
	},
	"copy 2": {
		{
			ArchivePath: "copy 2",
			Path:        "a/b/c",
			Name:        "f.txt",
			Hash:        "hhhh",
		},
		{
			ArchivePath: "copy 2",
			Path:        "a/b/e",
			Name:        "x.txt",
			Hash:        "gggg",
		},
		{
			ArchivePath: "copy 2",
			Path:        "a/b/e",
			Name:        "g.txt",
			Hash:        "tttt",
		},
		{
			ArchivePath: "copy 2",
			Path:        "",
			Name:        "x",
			Hash:        "asdfg",
		},
		{
			ArchivePath: "copy 2",
			Path:        "q/w/e/r/t",
			Name:        "y.txt",
			Hash:        "12345",
		},
		{
			ArchivePath: "copy 2",
			Path:        "",
			Name:        "yyy.txt",
			Hash:        "yyyy",
		},
	},
}
