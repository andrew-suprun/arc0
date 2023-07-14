package file_fs

import (
	m "arch/model"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func (s *scanner) handleFiles(cmd m.HandleFiles) {

	for _, delete := range cmd.Delete {
		log.Printf("delete: %s", delete)
		err := os.Remove(delete.String())
		if err != nil {
			s.events <- m.Error{Name: delete.Name, Error: err}
		}
		path := filepath.Join(delete.Root.String(), delete.Path.String())
		fsys := os.DirFS(path)

		entries, _ := fs.ReadDir(fsys, ".")
		hasFiles := false
		for _, entry := range entries {
			if entry.Name() != ".DS_Store" && !strings.HasPrefix(entry.Name(), "._") {
				hasFiles = true
				break
			}
		}
		if !hasFiles {
			os.RemoveAll(path)
		}
	}

	for _, rename := range cmd.Rename {
		path := filepath.Join(rename.Root.String(), rename.NewId.Path.String())
		err := os.MkdirAll(path, 0755)
		if err != nil {
			s.events <- m.Error{Name: rename.Name, Error: err}
		}
		err = os.Rename(rename.Id.String(), rename.NewId.String())
		if err != nil {
			s.events <- m.Error{Name: rename.Name, Error: err}
		}
	}

	if cmd.Copy != nil {
		log.Printf("copy: from: %q, to: %q", cmd.Copy.From, cmd.Copy.To)
		err := s.copyFiles(cmd.Copy.From, cmd.Copy.To)
		if err != nil {
			s.events <- m.Error{Name: cmd.Copy.From.Name, Error: err}
		}
	}
	s.events <- m.FilesHandled(cmd)
}

type command interface {
	cmd()
}

type createFile m.Id

func (createFile) cmd() {}

type writeBuffer []byte

func (writeBuffer) cmd() {}

type event interface {
	event()
}

type copyProgress uint64

func (copyProgress) event() {}

type copyError m.Error

func (copyError) event() {}

func (s *scanner) copyFiles(source m.Id, targets []m.Root) error {
	events := make([]chan event, len(targets))
	copied := make([]uint64, len(targets))
	reported := uint64(0)

	for i := range targets {
		events[i] = make(chan event)
	}

	go s.reader(source, targets, events)

	for {
		hasValue := false
		for i := range events {
			if event, ok := <-events[i]; ok {
				hasValue = true
				switch event := event.(type) {
				case copyProgress:
					copied[i] += uint64(event)

				case copyError:
					s.events <- m.Error{Name: event.Name, Error: event.Error}
				}
			}
		}
		minCopied := uint64(0)
		for _, fileCopied := range copied {
			if minCopied > fileCopied {
				minCopied = fileCopied
			}
		}
		if reported < minCopied {
			reported = minCopied
			s.events <- m.CopyingProgress(reported)
		}
		if !hasValue {
			break
		}
	}

	return nil
}

func (s *scanner) reader(source m.Id, targets []m.Root, eventChans []chan event) {
	commands := make([]chan command, len(targets))
	defer func() {
		for _, cmdChan := range commands {
			close(cmdChan)
		}
	}()

	for i := range targets {
		commands[i] = make(chan command)
		go s.writer(commands[i], eventChans[i])
	}

	for i, root := range targets {
		commands[i] <- createFile(m.Id{Root: root, Name: source.Name})
	}

	sourceFile, err := os.Open(source.String())
	if err != nil {
		s.events <- m.Error{Name: source.Name, Error: err}
		return
	}

	var n int
	for err != io.EOF {
		buf := make([]byte, 1024*1024)
		n, err = sourceFile.Read(buf)
		if err != nil && err != io.EOF {
			s.events <- m.Error{Name: source.Name, Error: err}
			return
		}
		for _, cmd := range commands {
			cmd <- writeBuffer(buf[:n])
		}
	}
}

func (s *scanner) writer(cmdChan chan command, eventChan chan event) {
	var id m.Id
	var file *os.File
	var err error
	var copied copyProgress

	defer func() {
		if file != nil {
			file.Close()
		}
		close(eventChan)
	}()

	for cmd := range cmdChan {
		switch cmd := cmd.(type) {
		case createFile:
			os.MkdirAll(filepath.Join(cmd.Root.String(), cmd.Path.String()), 0755)
			id = m.Id(cmd)
			file, err = os.Create(id.String())

		case writeBuffer:
			n, err := file.Write([]byte(cmd))
			copied += copyProgress(n)
			if err != nil {
				s.events <- m.Error{Name: id.Name, Error: err}
				return
			}
			eventChan <- copied

		}
		if err != nil {
			eventChan <- copyError{Name: id.Name, Error: err}
		}
	}
}
