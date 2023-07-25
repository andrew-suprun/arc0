package controller

import (
	m "arch/model"
	"log"
	"testing"
	"time"
)

func TestResolveName(t *testing.T) {
	c := newController([]m.Root{"origin", "copy1", "copy2"})
	c.archives["origin"] = &archive{
		scanner: &testScanner{},
	}
	c.archiveScanned(m.ArchiveScanned{
		Root: "origin",
		Files: []m.Meta{
			{
				Id:      m.Id{Root: "origin", Name: m.Name{Path: "x", Base: "y"}},
				Size:    1,
				ModTime: time.Now(),
				Hash:    "x/y",
			},
			{
				Id:      m.Id{Root: "origin", Name: m.Name{Path: "x", Base: "y [1]"}},
				Size:    1,
				ModTime: time.Now(),
				Hash:    "x/y [1]",
			},
		},
	})
	newName, updated := c.resolveName(m.Name{Path: "x/y", Base: "z"})
	log.Printf("TestResolveName: new name: %q, updated: %v", newName, updated)
	newName, updated = c.resolveName(m.Name{Path: "x/y/z", Base: "aaa"})
	log.Printf("TestResolveName: new name: %q, updated: %v", newName, updated)
}

func TestNameConflict(t *testing.T) {
	c := newController([]m.Root{"origin", "copy1", "copy2"})
	c.archives["origin"] = &archive{
		scanner: &testScanner{},
	}
	c.archiveScanned(m.ArchiveScanned{
		Root: "origin",
		Files: []m.Meta{
			{
				Id:      m.Id{Root: "origin", Name: m.Name{Path: "x", Base: "y"}},
				Size:    1,
				ModTime: time.Now(),
				Hash:    "x/y",
			},
			{
				Id:      m.Id{Root: "origin", Name: m.Name{Path: "x", Base: "y [1]"}},
				Size:    1,
				ModTime: time.Now(),
				Hash:    "x/y [1]",
			},
		},
	})
	c.addCopyFile(&m.File{
		Meta: m.Meta{
			Id:      m.Id{Root: "copy1", Name: m.Name{Path: "x/y", Base: "z"}},
			Size:    1,
			ModTime: time.Now(),
			Hash:    "x/y/z",
		},
	})
	c.addCopyFile(&m.File{
		Meta: m.Meta{
			Id:      m.Id{Root: "copy2", Name: m.Name{Path: "x", Base: "y [1]"}},
			Size:    1,
			ModTime: time.Now(),
			Hash:    "x/y [1]",
		},
	})
	c.addCopyFile(&m.File{
		Meta: m.Meta{
			Id:      m.Id{Root: "copy2", Name: m.Name{Path: "x", Base: "y [2]"}},
			Size:    1,
			ModTime: time.Now(),
			Hash:    "x/y [2]",
		},
	})
	for path, folder := range c.folders {
		log.Printf("FOLDER: %q", path)
		for _, entry := range folder.sort() {
			log.Printf("    %s", entry)
		}
	}
	log.Printf("CMDS:")
	for _, cmd := range cmds {
		log.Printf("    %#v", cmd)
	}
}

var cmds []m.FileCommand

type testScanner struct{}

func (s *testScanner) Send(cmd m.FileCommand) {
	cmds = append(cmds, cmd)
}
