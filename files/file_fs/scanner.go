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

	"golang.org/x/text/unicode/norm"
)

type scanner struct {
	events      events.EventChan
	lc          *lifecycle.Lifecycle
	archivePath string
	infos       map[uint64]*fileInfo
	totalSize   uint64
	totalHashed uint64
}

type fileInfo struct {
	meta events.FileMeta
	hash string
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

		fileMeta := events.FileMeta{
			INode:       sys.Ino,
			ArchivePath: s.archivePath,
			Path:        dir(path),
			Name:        filepath.Base(path),
			ModTime:     modTime,
			Size:        uint64(meta.Size()),
		}

		s.infos[sys.Ino] = &fileInfo{
			meta: fileMeta,
		}
		s.events <- fileMeta
		s.totalSize += fileMeta.Size

		return nil
	})
}

func (s *scanner) hashArchive() {
	s.lc.Started()
	defer s.lc.Done()

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

	fileInfos := make([]*fileInfo, 0, len(s.infos))
	for _, info := range s.infos {
		fileInfos = append(fileInfos, info)
	}

	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].meta.Size < fileInfos[j].meta.Size
	})

	for i, info := range fileInfos {
		if info.hash == "" {
			if s.lc.ShoudStop() {
				return
			}
			s.hashFile(fileInfos[i])

			s.events <- events.FileHash{
				INode:       info.meta.INode,
				ArchivePath: info.meta.ArchivePath,
				Hash:        info.hash,
			}
		}
	}
}

func (s *scanner) hashFile(info *fileInfo) {
	hash := sha256.New()
	buf := make([]byte, 1024*1024)

	fsys := os.DirFS(s.archivePath)
	file, err := fsys.Open(filepath.Join(info.meta.Path, info.meta.Name))
	if err != nil {
		s.events <- events.ScanError{Meta: info.meta, Error: err}
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
					s.events <- events.ScanError{Meta: info.meta, Error: err}
					return
				}
			}
			if nr != nw {
				s.events <- events.ScanError{Meta: info.meta, Error: err}
				return
			}
		}

		if er == io.EOF {
			break
		}
		if er != nil {
			s.events <- events.ScanError{Meta: info.meta, Error: er}
			return
		}

		s.totalHashed += uint64(nr)
		s.events <- events.ScanProgress{
			ArchivePath:  s.archivePath,
			ScanState:    events.HashFileTree,
			ScanProgress: float64(s.totalHashed) / float64(s.totalSize),
		}
	}
	info.hash = base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}

func (s *scanner) readMeta() {
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
		if len(record) == 5 {
			iNode, er1 := strconv.ParseUint(record[0], 10, 64)
			size, er2 := strconv.ParseUint(record[2], 10, 64)
			modTime, er3 := time.Parse(time.RFC3339, record[3])
			modTime = modTime.UTC().Round(time.Second)
			hash := record[4]
			if hash == "" || er1 != nil || er2 != nil || er3 != nil {
				continue
			}

			info, ok := s.infos[iNode]
			if hash != "" && ok && info.meta.ModTime == modTime && info.meta.Size == size {
				info.hash = hash
				s.events <- events.FileHash{
					INode:       iNode,
					ArchivePath: s.archivePath,
					Hash:        hash,
				}
				s.totalHashed += info.meta.Size
				s.events <- events.ScanProgress{
					ArchivePath:  s.archivePath,
					ScanState:    events.HashFileTree,
					ScanProgress: float64(s.totalHashed) / float64(s.totalSize),
				}
			}
		}
	}
}

func (s *scanner) storeMeta() error {
	result := make([][]string, 1, len(s.infos)+1)
	result[0] = []string{"INode", "Name", "Size", "ModTime", "Hash"}

	for iNode, info := range s.infos {
		result = append(result, []string{
			fmt.Sprint(iNode),
			norm.NFC.String(filepath.Join(info.meta.Path, info.meta.Name)),
			fmt.Sprint(info.meta.Size),
			info.meta.ModTime.UTC().Format(time.RFC3339Nano),
			info.hash,
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
