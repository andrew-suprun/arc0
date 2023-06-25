package mock_fs

import (
	"arch/actor"
	"arch/model"
	"log"
	"math/rand"
	"strings"
	"time"
)

var Scan bool

type mockFs struct {
	events model.EventChan
}

type scanner struct {
	root   string
	events model.EventChan
}

func NewFs(events model.EventChan) model.FS {
	return &mockFs{events: events}
}

func (fs *mockFs) NewArchiveScanner(root string) model.ArchiveScanner {
	return &scanner{root: root, events: fs.events}
}

func (fs *mockFs) NewFileHandler() actor.Actor[model.HandleFiles] {
	return actor.NewActor[model.HandleFiles](fs.handleFiles)
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
			INode:   meta.INode,
			Root:    s.root,
			Name:    meta.Name,
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
				Root: meta.Root,
				Name: meta.Name,
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
				Root: meta.Root,
				Name: meta.Name,
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
	log.Printf("### handleFiles: msg=%v", msg)
	if msg.Copy != nil {
		for _, meta := range metas[msg.Copy.SourceRoot] {
			if meta.Name == msg.Copy.Name {
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

var sizes = map[string]uint64{}
var modTimes = map[string]time.Time{}
var inode = uint64(0)

func init() {
	sizes["yyyy"] = 50000000
	sizes["hhhh"] = 50000000
	for root, metaStrings := range metaMap {
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
var metaMap = map[string][]string{
	"origin": {
		"xxx.txt:xxxx",
		"a/b/c/x.txt:hhhh",
		"a/b/e/f.txt:gggg",
		"a/b/e/g.txt:tttt",
		"x.txt:hhhh",
		"q/w/e/r/t/y.txt:qwerty",
		"yyy.txt:yyyy",
		"0000:0000",
	},
	"copy 1": {
		"xxx.txt:xxxx",
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
		"3333:3333",
	},
	"copy 2": {
		"xxx.txt:xxxx",
		"a/b/c/f.txt:hhhh",
		"a/b/e/x.txt:gggg",
		"a/b/e/g.txt:tttt",
		"x:asdfg",
		"q/w/e/r/t/y.txt:12345",
		"2222:0000",
		"3333:3333",
	},
}
