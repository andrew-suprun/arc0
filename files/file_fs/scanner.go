package file_fs

import (
	"arch/lifecycle"
	m "arch/model"
	"arch/stream"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/text/unicode/norm"
)

const hashFileName = ".meta.csv"

type scanner struct {
	root     m.Root
	events   *stream.Stream[m.Event]
	commands *stream.Stream[m.FileCommand]
	lc       *lifecycle.Lifecycle
	byInode  map[uint64]*m.Meta
	files    []m.Meta
	stored   map[uint64]*m.Meta
	sent     map[m.Id]struct{}
}

func (s *scanner) Send(cmd m.FileCommand) {
	s.commands.Push(cmd)
}

func (s *scanner) handleEvents() {
	for {
		for _, cmd := range s.commands.Pull() {
			s.handleCommand(cmd)
		}
	}
}

func (s *scanner) handleCommand(cmd m.FileCommand) {
	s.lc.Started()
	defer s.lc.Done()

	switch cmd := cmd.(type) {
	case m.ScanArchive:
		s.scanArchive()

	case m.HashArchive:
		s.hashArchive()

	case m.DeleteFile:
		s.deleteFile(cmd)

	case m.RenameFile:
		s.renameFile(cmd)

	case m.CopyFile:
		s.copyFile(cmd)
	}
}

func (s *scanner) scanArchive() {
	fsys := os.DirFS(s.root.String())
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if s.lc.ShoudStop() || !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		if err != nil {
			s.events.Push(m.Error{
				Id:    m.Id{Root: s.root, Name: m.Path(path).ParentName()},
				Error: err})
			return nil
		}

		meta, err := d.Info()
		if err != nil {
			s.events.Push(m.Error{
				Id:    m.Id{Root: s.root, Name: m.Path(path).ParentName()},
				Error: err})
			return nil
		}
		sys := meta.Sys().(*syscall.Stat_t)
		modTime := meta.ModTime()
		modTime = modTime.UTC().Round(time.Second)

		file := &m.Meta{
			Id:      m.Id{Root: s.root, Name: m.Path(path).ParentName()},
			ModTime: modTime,
			Size:    uint64(meta.Size()),
		}

		s.byInode[sys.Ino] = file
		s.files = append(s.files, *file)
		log.Printf("file: %q", file.Id)

		return nil
	})

	s.events.Push(m.ArchiveScanned{
		Root:  s.root,
		Files: s.files,
	})
}

func (s *scanner) hashArchive() {

	s.readMeta()
	defer func() {
		s.storeMeta()
	}()

	for ino, file := range s.byInode {
		if stored, ok := s.stored[ino]; ok && stored.ModTime == file.ModTime && stored.Size == file.Size {
			file.Hash = stored.Hash
			s.events.Push(m.FileHashed{Id: file.Id, Hash: file.Hash})
			s.sent[file.Id] = struct{}{}
		}
	}

	for _, file := range s.files {
		if _, ok := s.sent[file.Id]; ok {
			continue
		}

		file.Hash = s.hashFile(file.Id)

		if s.lc.ShoudStop() {
			return
		}

		s.events.Push(m.FileHashed{Id: file.Id, Hash: file.Hash})
	}
}

func (s *scanner) hashFile(id m.Id) m.Hash {
	hash := sha256.New()
	buf := make([]byte, 1024*1024)
	var hashed uint64

	fsys := os.DirFS(id.Root.String())
	file, err := fsys.Open(id.Name.String())
	if err != nil {
		s.events.Push(m.Error{Id: id, Error: err})
		return ""
	}
	defer file.Close()

	for {
		if s.lc.ShoudStop() {
			return ""
		}

		nr, er := file.Read(buf)
		if nr > 0 {
			nw, ew := hash.Write(buf[0:nr])
			if ew != nil {
				if err != nil {
					s.events.Push(m.Error{Id: id, Error: err})
					return ""
				}
			}
			if nr != nw {
				s.events.Push(m.Error{Id: id, Error: err})
				return ""
			}
		}

		if er == io.EOF {
			break
		}
		if er != nil {
			s.events.Push(m.Error{Id: id, Error: er})
			return ""
		}

		hashed += uint64(nr)
		s.events.Push(m.HashingProgress{
			Root:   id.Root,
			Hashed: hashed,
		})
	}
	return m.Hash(base64.RawURLEncoding.EncodeToString(hash.Sum(nil)))
}

func (s *scanner) readMeta() {
	absHashFileName := filepath.Join(s.root.String(), hashFileName)
	hashInfoFile, err := os.Open(absHashFileName)
	if err != nil {
		return
	}
	defer hashInfoFile.Close()

	records, err := csv.NewReader(hashInfoFile).ReadAll()
	if err != nil || len(records) == 0 {
		return
	}

	for _, record := range records[1:] {
		if len(record) == 5 {
			iNode, er1 := strconv.ParseUint(record[0], 10, 64)
			size, er2 := strconv.ParseUint(record[2], 10, 64)
			modTime, er3 := time.Parse(time.RFC3339, record[3])
			modTime = modTime.UTC().Round(time.Second)
			hash := record[4]
			if hash == "" || er1 != nil || er2 != nil || er3 != nil {
				continue
			}

			s.stored[iNode] = &m.Meta{
				ModTime: modTime,
				Size:    uint64(size),
				Hash:    m.Hash(hash),
			}
			info, ok := s.byInode[iNode]
			if hash != "" && ok && info.ModTime == modTime && info.Size == size {
				info.Hash = m.Hash(hash)
			}
		}
	}
}

func (s *scanner) storeMeta() error {
	result := make([][]string, 1, len(s.byInode)+1)
	result[0] = []string{"INode", "Name", "Size", "ModTime", "Hash"}

	for iNode, file := range s.byInode {
		result = append(result, []string{
			fmt.Sprint(iNode),
			norm.NFC.String(file.Name.String()),
			fmt.Sprint(file.Size),
			file.ModTime.UTC().Format(time.RFC3339Nano),
			file.Hash.String(),
		})
	}

	absHashFileName := filepath.Join(s.root.String(), hashFileName)
	hashInfoFile, err := os.Create(absHashFileName)

	if err != nil {
		return err
	}
	err = csv.NewWriter(hashInfoFile).WriteAll(result)
	hashInfoFile.Close()
	return err
}
