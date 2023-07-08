package mock_fs

import (
	"arch/actor"
	m "arch/model"
	"math/rand"
	"path/filepath"
	"time"
)

var Scan bool

type mockFs struct {
	events m.EventChan
}

type scanner struct {
	root   m.Root
	events m.EventChan
	actor.Actor[m.FileCommand]
}

func NewFs(events m.EventChan) m.FS {
	fs := &mockFs{events: events}
	return fs
}

func (fs *mockFs) NewArchiveScanner(root m.Root) m.ArchiveScanner {
	s := &scanner{
		root:   root,
		events: fs.events,
	}
	s.Actor = actor.NewActor[m.FileCommand](s.handleFiles)
	return s
}

func (s *scanner) handleFiles(cmd m.FileCommand) bool {
	switch cmd := cmd.(type) {
	case m.ScanArchive:
		s.scanArchive()

	case m.HashArchive:
		s.hashArchive()

	case m.DeleteFile:
		s.events <- m.FileDeleted(cmd)

	case m.RenameFile:
		s.events <- m.FileRenamed(cmd)

	case m.CopyFile:
		for _, meta := range metas[cmd.From.Root] {
			if meta.FullName == cmd.From.FullName().String() {
				for copied := uint64(0); ; copied += 50000 {
					if copied > meta.Size {
						copied = meta.Size
					}
					s.events <- m.Progress{
						Root:          cmd.To,
						ProgressState: m.CopyingFile,
						HandledSize:   copied,
					}
					if copied == meta.Size {
						break
					}
					time.Sleep(time.Millisecond)
				}
				break
			}
		}
		s.events <- m.FileCopied(cmd)
	}
	return true
}

func (s *scanner) scanArchive() {
	archFiles := metas[s.root]
	var archiveMetas m.FileMetas
	for _, meta := range archFiles {
		archiveMetas = append(archiveMetas, m.FileMeta{
			FileId: m.FileId{
				Root: s.root,
				Path: dir(meta.FullName),
				Name: name(meta.FullName),
			},
			Size:    meta.Size,
			ModTime: meta.ModTime,
		})
	}
	s.events <- m.ArchiveScanned{
		Root:      s.root,
		FileMetas: archiveMetas,
	}
}

func (s *scanner) hashArchive() {
	archFiles := metas[s.root]
	scans := make([]bool, len(archFiles))

	for i := range archFiles {
		scans[i] = Scan && rand.Intn(2) == 0
	}
	for i := range archFiles {
		if !scans[i] {
			meta := archFiles[i]
			s.events <- m.FileHashed{
				FileId: m.FileId{
					Root: meta.Root,
					Path: dir(meta.FullName),
					Name: name(meta.FullName),
				},
				Hash: meta.Hash,
			}
		}
	}
	for i := range archFiles {
		if scans[i] {
			meta := archFiles[i]
			for hashed := uint64(0); ; hashed += 50000 {
				if hashed > meta.Size {
					hashed = meta.Size
				}
				s.events <- m.Progress{
					Root:          meta.Root,
					ProgressState: m.HashingFile,
					HandledSize:   hashed,
				}
				if hashed == meta.Size {
					break
				}
				time.Sleep(time.Millisecond)
			}
			s.events <- m.FileHashed{
				FileId: m.FileId{
					Root: meta.Root,
					Path: dir(meta.FullName),
					Name: name(meta.FullName),
				},
				Hash: meta.Hash,
			}
		}
	}
	s.events <- m.Progress{
		Root:          s.root,
		ProgressState: m.FileTreeHashed,
	}
}

var beginning = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
var end = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
var duration = end.Sub(beginning)

type fileMeta struct {
	INode uint64
	m.Root
	FullName string
	m.Hash
	Size    uint64
	ModTime time.Time
}

var sizeByName = map[string]uint64{}
var sizeByHash = map[m.Hash]uint64{}
var modTimes = map[m.Hash]time.Time{}
var inode = uint64(0)

func init() {
	sizeByName["8888"] = 88888888
	sizeByHash["yyyy"] = 50000000
	sizeByHash["hhhh"] = 50000000
	for root, metaStrings := range metaMap {
		for name, hash := range metaStrings {
			size, ok := sizeByName[name]
			if !ok {
				size, ok = sizeByHash[hash]
				if !ok {
					size = uint64(rand.Intn(100000000))
					sizeByHash[hash] = size
				}
			}
			modTime, ok := modTimes[hash]
			if !ok {
				modTime = beginning.Add(time.Duration(rand.Int63n(int64(duration))))
				modTimes[hash] = modTime
			}
			inode++
			file := &fileMeta{
				INode:    inode,
				Root:     root,
				FullName: name,
				Hash:     hash,
				Size:     size,
				ModTime:  modTime,
			}
			metas[root] = append(metas[root], file)
		}
	}
}

var metas = map[m.Root][]*fileMeta{}
var metaMap = map[m.Root]map[string]m.Hash{
	"origin": {
		// "0000":            "0000",
		"6666": "6666",
		"7777": "7777",
		// "a/b/e/f.txt":     "gggg",
		// "a/b/e/g.txt":     "tttt",
		// "x/xxx.txt":       "hhhh",
		// "q/w/e/r/t/y.txt": "qwerty",
		// "qqq.txt":         "hhhh",
		// "xxx.txt":         "xxxx",
		// "yyy.txt":         "yyyy",
	},
	"copy 1": {
		// "xxx.txt":     "xxxx",
		// "a/b/c/d.txt": "llll",
		// "a/b/e/f.txt": "hhhh",
		// "a/b/e/g.txt": "tttt",
		// "qqq.txt":     "mmmm",
		// "y.txt":       "gggg",
		// "x/xxx.txt":   "hhhh",
		// "zzz.txt":     "hhhh",
		// "x/y/z.txt":   "zzzz",
		// "yyy.txt":     "yyyy",
		// "1111":        "0000",
		// "3333":        "3333",
		// "4444":        "4444",
		// "8888":        "9999",
		// "b/bbb.txt":   "bbbb",
		"6666": "6666",
		"7777": "7777",
	},
	"copy 2": {
		// "xxx.txt":         "xxxx",
		// "a/b/c/f.txt":     "hhhh",
		// "a/b/e/x.txt":     "gggg",
		// "a/b/e/g.txt":     "tttt",
		// "x":               "asdfg",
		// "q/w/e/r/t/y.txt": "12345",
		// "2222":            "0000",
		// "3333":            "3333",
		// "5555":            "4444",
		"6666": "7777",
		"7777": "6666",
		// "8888":            "8888",
		// "c/ccc.txt":       "bbbb",
	},
}

func dir(path string) m.Path {
	path = filepath.Dir(path)
	if path == "." {
		return ""
	}
	return m.Path(path)
}

func name(path string) m.Name {
	return m.Name(filepath.Base(path))
}
