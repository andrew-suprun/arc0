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
	root        string
	totalSize   uint64
	totalHashed uint64
}

func (fsys *mockFs) NewScanner(root string) model.Scanner {
	return &scanner{
		scan:   fsys.scan,
		events: fsys.events,
		root:   root,
	}
}

func (s *scanner) Handler(msg model.Msg) bool {
	switch msg := msg.(type) {
	case model.ScanArchive:
		return s.scanArchive()
	case model.HashArchive:
		return s.hashArchive()
	case model.CopyFile:
		return s.copy(msg)
	case model.RenameFile:
		return s.rename(msg)
	case model.DeleteFile:
		return s.delete(msg)
	}
	log.Panicf("### ERROR: Unhandled scanner message: %#v", msg)
	return false
}

func (s *scanner) scanArchive() bool {
	archFiles := metas[s.root]
	for _, meta := range archFiles {
		s.events <- model.FileScanned{
			INode:   meta.INode,
			Root:    s.root,
			Name:    meta.Name,
			Size:    meta.Size,
			ModTime: meta.ModTime,
		}
		s.totalSize += meta.Size
	}
	s.events <- model.Progress{
		Root:          s.root,
		ProgressState: model.WalkingFileTreeComplete,
	}
	return true
}

func (s *scanner) hashArchive() bool {
	archFiles := metas[s.root]
	scans := make([]bool, len(archFiles))

	for i := range archFiles {
		scans[i] = s.scan && rand.Intn(2) == 0
	}
	for i := range archFiles {
		if !scans[i] {
			meta := archFiles[i]
			s.events <- model.FileHashed{
				INode: meta.INode,
				Root:  meta.Root,
				Hash:  meta.Hash,
			}
			s.totalHashed += meta.Size
			s.events <- model.Progress{
				Root:          s.root,
				ProgressState: model.HashingFileTree,
				Processed:     s.totalHashed,
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
				s.events <- model.Progress{
					Root:          meta.Root,
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
				INode: meta.INode,
				Root:  meta.Root,
				Hash:  meta.Hash,
			}
		}
	}
	s.events <- model.Progress{
		Root:          s.root,
		ProgressState: model.HashingFileTreeComplete,
	}
	return true
}

func (s *scanner) copy(msg model.CopyFile) bool {
	log.Printf("### scanner copy: arch=%q: from %q/%x", s.root, msg.Root, msg.INode)
	var size uint64
	var meta *fileMeta
	for _, meta = range metas[msg.Root] {
		if meta.INode == msg.INode {
			size = meta.Size
			break
		}
	}

	for copied := uint64(0); ; copied += 10000 {
		if copied > size {
			copied = size
		}
		s.events <- model.Progress{
			Root:          s.root,
			ProgressState: model.CopyingFile,
			Processed:     copied,
		}
		if copied == size {
			break
		}
		time.Sleep(time.Millisecond)
	}
	log.Printf("### mockFs.Copied: root=%q, from=%q, ino=%x, size=%d", s.root, msg.Root, msg.INode, size)
	s.events <- model.FileCopied{
		FromRoot: msg.Root,
		ToRoot:   s.root,
		Name:     meta.Name,
	}
	return true
}

func (s *scanner) rename(msg model.RenameFile) bool {
	// log.Printf("### scanner move: arch=%q from %#v to %#v", s.root, from.AbsName(), to.AbsName())
	s.events <- model.FileRenamed{
		Root:    s.root,
		INode:   msg.INode,
		NewName: msg.NewName,
	}
	return true
}

func (s *scanner) delete(msg model.DeleteFile) bool {
	// log.Printf("### scanner delete: arch=%q file %#v", s.root, meta.AbsName())
	s.events <- model.FileDeleted{
		Root:  s.root,
		INode: msg.INode,
	}
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
		"3333:3333",
	},
	"copy 2": {
		"a/b/c/f.txt:hhhh",
		"a/b/e/x.txt:gggg",
		"a/b/e/g.txt:tttt",
		"x:asdfg",
		"q/w/e/r/t/y.txt:12345",
		"2222:0000",
		"3333:3333",
	},
}
