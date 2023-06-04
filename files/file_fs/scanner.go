package file_fs

import (
	"arch/events"
	"arch/lifecycle"
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
	"time"

	"syscall"
)

type scanner struct {
	events      events.EventChan
	lc          *lifecycle.Lifecycle
	archivePath string
	metas       []*events.FileMeta
	byIno       map[uint64]*events.FileMeta
	hashes      map[uint64]string
}

func (scanner *scanner) ScanArchive() {
	go scanner.scanArchive()
}

func (scanner *scanner) HashArchive() {
	go scanner.hashArchive()
}

const hashFileName = ".meta.csv"

func (s *scanner) scanArchive() {
	s.lc.Started()
	defer s.lc.Done()

	defer func() {
		s.events <- events.ScanProgress{
			ArchivePath: s.archivePath,
			ScanState:   events.WalkFileTreeComplete,
		}
	}()

	fsys := os.DirFS(s.archivePath)
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if s.lc.ShoudStop() || !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		if err != nil {
			s.events <- events.ScanError{
				Meta: events.FileMeta{
					ArchivePath: s.archivePath,
					Path:        dir(path),
					Name:        filepath.Base(path),
				},
				Error: err}
			return nil
		}

		meta, err := d.Info()
		if err != nil {
			s.events <- events.ScanError{
				Meta: events.FileMeta{
					ArchivePath: s.archivePath,
					Path:        dir(path),
					Name:        filepath.Base(path),
				},
				Error: err}
			return nil
		}
		sys := meta.Sys().(*syscall.Stat_t)
		modTime := meta.ModTime()
		modTime = modTime.UTC().Round(time.Second)

		fileMeta := &events.FileMeta{
			Ino:         sys.Ino,
			ArchivePath: s.archivePath,
			Path:        dir(path),
			Name:        filepath.Base(path),
			ModTime:     modTime,
			Size:        uint64(meta.Size()),
		}

		s.metas = append(s.metas, fileMeta)
		s.byIno[sys.Ino] = fileMeta
		s.events <- *fileMeta

		return nil
	})
}

func (s *scanner) hashArchive() {
	s.readMeta()
	defer func() {
		s.storeMeta()
	}()

	defer func() {
		s.events <- events.ScanProgress{
			ArchivePath: s.archivePath,
			ScanState:   events.HashFileTreeComplete,
		}
	}()

	s.byIno = map[uint64]*events.FileMeta{}
	s.hashes = map[uint64]string{}

	sort.Slice(s.metas, func(i, j int) bool {
		return s.metas[i].Size < s.metas[j].Size
	})

	for _, meta := range s.metas {
		if s.hashes[meta.Ino] == "" {
			if s.lc.ShoudStop() {
				return
			}
			hash := s.hashFile(meta)
			s.hashes[meta.Ino] = hash
			s.events <- events.FileHash{
				Ino:         meta.Ino,
				ArchivePath: meta.ArchivePath,
				Hash:        hash,
			}
		}
	}
}

func (s *scanner) hashFile(meta *events.FileMeta) string {
	hash := sha256.New()
	buf := make([]byte, 256*1024)

	fsys := os.DirFS(s.archivePath)
	file, err := fsys.Open(filepath.Join(meta.Path, meta.Name))
	if err != nil {
		s.events <- events.ScanError{Meta: *meta, Error: err}
		return ""
	}
	defer file.Close()

	var hashed uint64

	for {
		if s.lc.ShoudStop() {
			return ""
		}

		nr, er := file.Read(buf)
		if nr > 0 {
			nw, ew := hash.Write(buf[0:nr])
			if ew != nil {
				if err != nil {
					s.events <- events.ScanError{Meta: *meta, Error: err}
					return ""
				}
			}
			if nr != nw {
				s.events <- events.ScanError{Meta: *meta, Error: err}
				return ""
			}
		}

		hashed += uint64(nr)

		if er == io.EOF {
			break
		}
		if er != nil {
			s.events <- events.ScanError{Meta: *meta, Error: er}
			return ""
		}

		s.events <- events.ScanProgress{
			ArchivePath: s.archivePath,
			ScanState:   events.HashFileTree,
			FileHashed:  hashed,
		}
	}
	return base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}

func (s *scanner) readMeta() {
	s.hashes = map[uint64]string{}
	absHashFileName := filepath.Join(s.archivePath, hashFileName)
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
		var ino uint64
		var hash string

		// TODO: 5 field records are depricated
		if len(record) == 5 {
			hash = record[4]
		} else if len(record) == 2 {
			hash = record[1]
		}
		ino, err := strconv.ParseUint(record[0], 10, 64)
		if err != nil {
			continue
		}
		if _, ok := s.byIno[ino]; ok {
			s.hashes[ino] = hash
			s.events <- events.FileHash{
				Ino:         ino,
				ArchivePath: s.archivePath,
				Hash:        hash,
			}
		}
	}
}

func (s *scanner) storeMeta() error {
	result := make([][]string, 1, len(s.metas)+1)
	result[0] = []string{"Inode", "Hash"}

	for ino, hash := range s.hashes {
		result = append(result, []string{
			fmt.Sprint(ino),
			hash,
		})
	}

	absHashFileName := filepath.Join(s.archivePath, hashFileName)
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
