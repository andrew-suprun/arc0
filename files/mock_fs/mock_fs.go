package mock_fs

import (
	"arch/actor"
	"arch/model"
	"math/rand"
	"path/filepath"
	"time"
)

var Scan bool

type mockFs struct {
	events  model.EventChan
	handler actor.Actor[model.HandleFiles]
}

type scanner struct {
	root   string
	events model.EventChan
}

func NewFs(events model.EventChan) model.FS {
	fs := &mockFs{events: events}
	fs.handler = actor.NewActor[model.HandleFiles](fs.handleFiles)
	return fs
}

func (fs *mockFs) NewArchiveScanner(root string) model.ArchiveScanner {
	return &scanner{root: root, events: fs.events}
}

func (fs *mockFs) Send(cmd model.HandleFiles) {
	fs.handler.Send(cmd)
}

func (s *scanner) ScanArchive() {
	go s.scanArchive()
}

func (s *scanner) HashArchive() {
	go s.hashArchive()
}

func (s *scanner) scanArchive() {
	archFiles := metas[s.root]
	var archiveMetas []model.FileMeta
	for _, meta := range archFiles {
		archiveMetas = append(archiveMetas, model.FileMeta{
			FileId: model.FileId{
				Root: s.root,
				Path: dir(meta.Name),
				Name: name(meta.Name),
			},
			Size:    meta.Size,
			ModTime: meta.ModTime,
		})
	}
	s.events <- model.ArchiveScanned{
		Root:  s.root,
		Metas: archiveMetas,
	}
}

func (s *scanner) hashArchive() {
	archFiles := metas[s.root]
	scans := make([]bool, len(archFiles))

	for i := range archFiles {
		scans[i] = Scan && rand.Intn(2) == 0
	}
	var totalHashed uint64
	for i := range archFiles {
		if !scans[i] {
			meta := archFiles[i]
			s.events <- model.FileHashed{
				FileId: model.FileId{
					Root: meta.Root,
					Path: dir(meta.Name),
					Name: name(meta.Name),
				},
				Hash: meta.Hash,
			}
			totalHashed += meta.Size
			s.events <- model.ScanProgress{
				Root:          s.root,
				ProgressState: model.HashingFileTree,
				TotalHashed:   totalHashed,
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
				s.events <- model.ScanProgress{
					Root:          meta.Root,
					ProgressState: model.HashingFileTree,
					TotalHashed:   totalHashed + hashed,
				}
				if hashed == meta.Size {
					break
				}
				time.Sleep(time.Millisecond)
			}
			totalHashed += meta.Size
			s.events <- model.FileHashed{
				FileId: model.FileId{
					Root: meta.Root,
					Path: dir(meta.Name),
					Name: name(meta.Name),
				},
				Hash: meta.Hash,
			}
		}
	}
	s.events <- model.ScanProgress{
		Root:          s.root,
		ProgressState: model.FileTreeHashed,
	}
}

func (fs *mockFs) handleFiles(msg model.HandleFiles) bool {
	if msg.Copy != nil {
		for _, meta := range metas[msg.Copy.Root] {
			if meta.Name == msg.Copy.FullName() {
				for copyed := uint64(0); ; copyed += 10000 {
					if copyed > meta.Size {
						copyed = meta.Size
					}
					fs.events <- model.FileCopyProgress(copyed)
					if copyed == meta.Size {
						break
					}
					time.Sleep(time.Millisecond)
				}
				break
			}
		}
	}
	fs.events <- model.FilesHandled(msg)
	return true
}

var beginning = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
var end = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
var duration = end.Sub(beginning)

type fileMeta struct {
	INode   uint64
	Root    string
	Name    string
	Hash    string
	Size    uint64
	ModTime time.Time
}

var sizeByName = map[string]uint64{}
var sizeByHash = map[string]uint64{}
var modTimes = map[string]time.Time{}
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
				INode:   inode,
				Root:    root,
				Name:    name,
				Hash:    hash,
				Size:    size,
				ModTime: modTime,
			}
			metas[root] = append(metas[root], file)
		}
	}
}

var metas = map[string][]*fileMeta{}
var metaMap = map[string]map[string]string{
	"origin": {
		"xxx.txt":         "xxxx",
		"a/b/c/x.txt":     "hhhh",
		"a/b/e/f.txt":     "gggg",
		"a/b/e/g.txt":     "tttt",
		"x.txt":           "hhhh",
		"q/w/e/r/t/y.txt": "qwerty",
		"yyy.txt":         "yyyy",
		"0000":            "0000",
		"6666":            "6666",
		"7777":            "7777",
	},
	"copy 1": {
		"xxx.txt":     "xxxx",
		"a/b/c/d.txt": "llll",
		"a/b/e/f.txt": "hhhh",
		"a/b/e/g.txt": "tttt",
		"x.txt":       "mmmm",
		"y.txt":       "gggg",
		"a/b/c/x.txt": "hhhh",
		"zzzz.txt":    "hhhh",
		"x/y/z.txt":   "zzzz",
		"yyy.txt":     "yyyy",
		"1111":        "0000",
		"3333":        "3333",
		"4444":        "4444",
		"8888":        "9999",
	},
	"copy 2": {
		"xxx.txt":         "xxxx",
		"a/b/c/f.txt":     "hhhh",
		"a/b/e/x.txt":     "gggg",
		"a/b/e/g.txt":     "tttt",
		"x":               "asdfg",
		"q/w/e/r/t/y.txt": "12345",
		"2222":            "0000",
		"3333":            "3333",
		"5555":            "4444",
		"6666":            "7777",
		"7777":            "6666",
		"8888":            "8888",
	},
}

func dir(path string) string {
	path = filepath.Dir(path)
	if path == "." {
		return ""
	}
	return path
}

func name(path string) string {
	return filepath.Base(path)
}
