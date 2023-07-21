package mock_fs

import (
	m "arch/model"
	"arch/stream"
	"math/rand"
	"path/filepath"
	"time"
)

var Scan bool

type mockFs struct {
	eventStream *stream.Stream[m.Event]
}

type scanner struct {
	root        m.Root
	eventStream *stream.Stream[m.Event]
	inEvents    *stream.Stream[m.FileCommand]
}

func NewFs(eventStream *stream.Stream[m.Event]) m.FS {
	fs := &mockFs{eventStream: eventStream}
	return fs
}

func (fs *mockFs) NewArchiveScanner(root m.Root) m.ArchiveScanner {
	s := &scanner{
		root:        root,
		eventStream: fs.eventStream,
		inEvents:    stream.NewStream[m.FileCommand](root.String()),
	}
	go s.handleEvents()
	return s
}

func (s *scanner) Send(cmd m.FileCommand) {
	s.inEvents.Push(cmd)
}

func (s *scanner) handleEvents() {
	for {
		s.handleEvent(s.inEvents.Pull())
	}
}

func (s *scanner) handleEvent(cmd m.FileCommand) {
	switch cmd := cmd.(type) {
	case m.ScanArchive:
		s.scanArchive()

	case m.DeleteFile:
		s.eventStream.Push(m.FileDeleted(cmd))

	case m.RenameFile:
		s.eventStream.Push(m.FileRenamed(cmd))
	case m.CopyFile:
		for _, meta := range metas[cmd.From.Root] {
			if meta.FullName == cmd.From.Name.String() {
				for copied := uint64(0); ; copied += 50000 {
					if copied > meta.Size {
						copied = meta.Size
					}
					s.eventStream.Push(m.CopyingProgress(copied))
					if copied == meta.Size {
						break
					}
					time.Sleep(time.Millisecond)
				}
				break
			}
		}
		s.eventStream.Push(m.FileCopied(cmd))
	}
}

func (s *scanner) scanArchive() {
	archFiles := metas[s.root]
	totalSize := uint64(0)
	for _, file := range archFiles {
		totalSize += file.Size
	}

	s.eventStream.Push(m.TotalSize{
		Root: s.root,
		Size: totalSize,
	})

	scans := make([]bool, len(archFiles))

	for i := range archFiles {
		scans[i] = Scan && rand.Intn(2) == 0
	}
	for i := range archFiles {
		if !scans[i] {
			meta := archFiles[i]
			s.eventStream.Push(m.FileScanned{
				File: &m.File{
					Id: m.Id{
						Root: meta.Root,
						Name: m.Name{
							Path: dir(meta.FullName),
							Base: name(meta.FullName),
						},
					},
					Size:    meta.Size,
					ModTime: meta.ModTime,
					Hash:    meta.Hash,
				},
			})
		}
	}
	for i := range archFiles {
		if scans[i] {
			meta := archFiles[i]
			for hashed := uint64(0); ; hashed += 50000 {
				if hashed > meta.Size {
					hashed = meta.Size
				}
				s.eventStream.Push(m.HashingProgress{Root: meta.Root, Hashed: hashed})
				if hashed == meta.Size {
					break
				}
				time.Sleep(time.Millisecond)
			}
			s.eventStream.Push(m.FileScanned{
				File: &m.File{
					Id: m.Id{
						Root: meta.Root,
						Name: m.Name{
							Path: dir(meta.FullName),
							Base: name(meta.FullName),
						},
					},
					Size:    meta.Size,
					ModTime: meta.ModTime,
					Hash:    meta.Hash,
				},
			})
		}
	}
	s.eventStream.Push(m.ArchiveScanned{Root: s.root})
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
		"0000":            "0000",
		"6666":            "6666",
		"7777":            "7777",
		"a/b/e/f.txt":     "gggg",
		"a/b/e/g.txt":     "tttt",
		"x/xxx.txt":       "hhhh",
		"q/w/e/r/t/y.txt": "qwerty",
		"qqq.txt":         "hhhh",
		"uuu.txt":         "hhhh",
		"xxx.txt":         "xxxx",
		"yyy.txt":         "yyyy",
		"same":            "same",
		"different":       "different",
	},
	"copy 1": {
		"xxx.txt":     "xxxx",
		"a/b/c/d.txt": "llll",
		"a/b/e/f.txt": "hhhh",
		"a/b/e/g.txt": "tttt",
		"qqq.txt":     "mmmm",
		"y.txt":       "gggg",
		"x/xxx.txt":   "hhhh",
		"zzz.txt":     "hhhh",
		"x/y/z.txt":   "zzzz",
		"yyy.txt":     "yyyy",
		"1111":        "0000",
		"9999":        "9999",
		"4444":        "4444",
		"8888":        "9999",
		"b/bbb.txt":   "bbbb",
		"6666":        "6666",
		"7777":        "7777",
		"same":        "same-copy",
		"different":   "different-copy1",
	},
	"copy 2": {
		"xxx.txt":         "xxxx",
		"a/b/e/x.txt":     "gggg",
		"a/b/e/g.txt":     "tttt",
		"x":               "asdfg",
		"q/w/e/r/t/y.txt": "12345",
		"2222":            "0000",
		"9999":            "9999",
		"5555":            "4444",
		"6666":            "7777",
		"7777":            "6666",
		"8888":            "8888",
		"c/ccc.txt":       "bbbb",
		"same":            "same-copy",
		"different":       "different-copy2",
	},
}

func dir(path string) m.Path {
	path = filepath.Dir(path)
	if path == "." {
		return ""
	}
	return m.Path(path)
}

func name(path string) m.Base {
	return m.Base(filepath.Base(path))
}
