package mock_fs

import (
	"arch/model"
	"log"
	"math/rand"
	"strings"
	"time"
)

type mockFs struct {
	scan   bool
	events model.EventChan
}

func NewFs(events model.EventChan, scan bool) model.FS {
	return &mockFs{
		scan:   scan,
		events: events,
	}
}

type scanner struct {
	scan        bool
	events      model.EventChan
	archivePath string
	totalSize   uint64
	totalHashed uint64
}

func (fsys *mockFs) NewScanner(archivePath string) model.Scanner {
	return &scanner{
		scan:        fsys.scan,
		events:      fsys.events,
		archivePath: archivePath,
	}
}

func (s *scanner) Handler(msg model.Msg) bool {
	switch msg := msg.(type) {
	case model.ScanArchive:
		return s.scanArchive()
	case model.HashArchive:
		return s.hashArchive()
	case model.CopyFile:
		return s.copy(model.FileMeta(msg))
	case model.RenameFile:
		return s.move(msg.OldMeta, msg.NewMeta)
	case model.DeleteFile:
		return s.delete(model.FileMeta(msg))
	}
	log.Panicf("### ERROR: Unhandled scanner message: %#v", msg)
	return false
}

func (s *scanner) scanArchive() bool {
	archFiles := metas[s.archivePath]
	for _, meta := range archFiles {
		s.events <- model.FileScanned{
			INode:       meta.INode,
			ArchivePath: s.archivePath,
			FullName:    meta.FullName,
			Size:        meta.Size,
			ModTime:     meta.ModTime,
		}
		s.totalSize += meta.Size
	}
	s.events <- model.Progress{
		ArchivePath:   s.archivePath,
		ProgressState: model.WalkingFileTreeComplete,
	}
	return true
}

func (s *scanner) hashArchive() bool {
	archFiles := metas[s.archivePath]
	scans := make([]bool, len(archFiles))

	for i := range archFiles {
		scans[i] = s.scan && rand.Intn(2) == 0
	}
	for i := range archFiles {
		if !scans[i] {
			meta := archFiles[i]
			s.events <- model.FileHashed{
				INode:       meta.INode,
				ArchivePath: meta.ArchivePath,
				Hash:        meta.Hash,
			}
			s.totalHashed += meta.Size
			s.events <- model.Progress{
				ArchivePath:   s.archivePath,
				ProgressState: model.HashingFileTree,
				Processed:     s.totalHashed,
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
				s.events <- model.Progress{
					ArchivePath:   meta.ArchivePath,
					ProgressState: model.HashingFileTree,
					Processed:     s.totalHashed + hashed,
				}
				if hashed == meta.Size {
					break
				}
				time.Sleep(time.Millisecond)
			}
			s.totalHashed += meta.Size
			s.events <- model.FileHashed{
				INode:       meta.INode,
				ArchivePath: meta.ArchivePath,
				Hash:        meta.Hash,
			}
		}
	}
	s.events <- model.Progress{
		ArchivePath:   s.archivePath,
		ProgressState: model.HashingFileTreeComplete,
	}
	return true
}

func (s *scanner) copy(meta model.FileMeta) bool {
	log.Printf("### scanner %q: copy from %#v", s.archivePath, meta.AbsName())
	for copied := uint64(0); ; copied += 10000 {
		if copied > meta.Size {
			copied = meta.Size
		}
		s.events <- model.Progress{
			ArchivePath:   s.archivePath,
			ProgressState: model.CopyingFile,
			Processed:     copied,
		}
		if copied == meta.Size {
			break
		}
		time.Sleep(time.Millisecond)
	}
	s.events <- model.FileCopied(meta)
	return true
}

func (s *scanner) move(from, to model.FileMeta) bool {
	log.Printf("### scanner %q: move from %#v to %#v", s.archivePath, from.AbsName(), to.AbsName())
	from.FullName = to.FullName
	s.events <- model.FileRenamed(from)
	return true
}

func (s *scanner) delete(meta model.FileMeta) bool {
	log.Printf("### scanner %q: delete file %#v", s.archivePath, meta.AbsName())
	s.events <- model.FileDeleted(meta)
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
		"0000:0000",
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
		"1111:0000",
	},
	"copy 2": {
		"a/b/c/f.txt:hhhh",
		"a/b/e/x.txt:gggg",
		"a/b/e/g.txt:tttt",
		"x:asdfg",
		"q/w/e/r/t/y.txt:12345",
		"2222:0000",
	},
}
