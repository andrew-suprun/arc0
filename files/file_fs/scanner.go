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
	"os"
	"path/filepath"
	"sort"
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
	inEvents *stream.Stream[m.FileCommand]
	lc       *lifecycle.Lifecycle
	files    map[uint64]*m.File
	stored   map[uint64]*m.File
	sent     map[m.Id]struct{}
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
	s.lc.Started()
	defer s.lc.Done()

	switch cmd := cmd.(type) {
	case m.ScanArchive:
		s.scanArchive()

	case m.DeleteFile:
		s.deleteFile(cmd)

	case m.RenameFile:
		s.renameFile(cmd)

	case m.CopyFile:
		s.copyFile(cmd)
	}
}

func (s *scanner) scanArchive() {
	defer func() {
		s.events.Push(m.ArchiveScanned{Root: s.root})
	}()

	totalSize := uint64(0)
	fsys := os.DirFS(s.root.String())
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if s.lc.ShoudStop() || !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		if err != nil {
			s.events.Push(m.Error{
				Id:    m.Id{Root: s.root, Name: m.Name{Path: m.Path(dir(path)), Base: m.Base(name(path))}},
				Error: err})
			return nil
		}

		meta, err := d.Info()
		if err != nil {
			s.events.Push(m.Error{
				Id:    m.Id{Root: s.root, Name: m.Name{Path: m.Path(dir(path)), Base: m.Base(name(path))}},
				Error: err})
			return nil
		}
		sys := meta.Sys().(*syscall.Stat_t)
		modTime := meta.ModTime()
		modTime = modTime.UTC().Round(time.Second)

		file := &m.File{
			Id: m.Id{
				Root: s.root,
				Name: m.Name{
					Path: m.Path(dir(path)),
					Base: m.Base(name(path)),
				},
			},
			ModTime: modTime,
			Size:    uint64(meta.Size()),
		}
		totalSize += file.Size

		s.files[sys.Ino] = file

		return nil
	})

	s.readMeta()
	defer func() {
		s.storeMeta()
	}()

	s.events.Push(m.TotalSize{
		Root: s.root,
		Size: totalSize,
	})

	for ino, file := range s.files {
		if stored, ok := s.stored[ino]; ok && stored.ModTime == file.ModTime && stored.Size == file.Size {
			file.Hash = stored.Hash
			s.events.Push(m.FileScanned{File: file})
			s.sent[file.Id] = struct{}{}
		}
	}

	files := []*m.File{}
	for _, file := range s.files {
		files = append(files, file)
	}

	sort.Slice(files, func(i, j int) bool {
		iName := strings.ToLower(files[i].Id.String())
		jName := strings.ToLower(files[j].Id.String())
		return iName < jName
	})

	for _, file := range files {
		if _, ok := s.sent[file.Id]; ok {
			continue
		}

		s.hashFile(file)

		if s.lc.ShoudStop() {
			return
		}

		s.events.Push(m.FileScanned{File: file})
	}
}

func (s *scanner) hashFile(info *m.File) {
	hash := sha256.New()
	buf := make([]byte, 1024*1024)
	var hashed uint64

	fsys := os.DirFS(info.Root.String())
	file, err := fsys.Open(info.Name.String())
	if err != nil {
		s.events.Push(m.Error{Id: info.Id, Error: err})
		return
	}
	defer file.Close()

	for {
		if s.lc.ShoudStop() {
			return
		}

		nr, er := file.Read(buf)
		if nr > 0 {
			nw, ew := hash.Write(buf[0:nr])
			if ew != nil {
				if err != nil {
					s.events.Push(m.Error{Id: info.Id, Error: err})
					return
				}
			}
			if nr != nw {
				s.events.Push(m.Error{Id: info.Id, Error: err})
				return
			}
		}

		if er == io.EOF {
			break
		}
		if er != nil {
			s.events.Push(m.Error{Id: info.Id, Error: er})
			return
		}

		hashed += uint64(nr)
		s.events.Push(m.HashingProgress{
			Root:   info.Root,
			Hashed: hashed,
		})
	}
	info.Hash = m.Hash(base64.RawURLEncoding.EncodeToString(hash.Sum(nil)))
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

			s.stored[iNode] = &m.File{
				ModTime: modTime,
				Size:    uint64(size),
				Hash:    m.Hash(hash),
			}
			info, ok := s.files[iNode]
			if hash != "" && ok && info.ModTime == modTime && info.Size == size {
				info.Hash = m.Hash(hash)
			}
		}
	}
}

func (s *scanner) storeMeta() error {
	result := make([][]string, 1, len(s.files)+1)
	result[0] = []string{"INode", "Name", "Size", "ModTime", "Hash"}

	for iNode, file := range s.files {
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
