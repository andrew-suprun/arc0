package file_fs

import (
	"arch/events"
	"arch/files"
	"arch/lifecycle"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"log"
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
	totalHashed uint64
}

type fileInfo struct {
	meta events.FileMeta
	hash string
}

func (s *scanner) Handler(msg files.Msg) bool {
	switch msg := msg.(type) {
	case files.ScanArchive:
		return s.scanArchive()
	case files.HashArchive:
		return s.hashArchive()
	case files.Copy:
		return s.copy(msg.Source)
	case files.Move:
		return s.move(msg.OldMeta, msg.NewMeta)
	case files.Delete:
		return s.remove(msg.File)
	}
	log.Panicf("### ERROR: Unhandled scanner message: %#v", msg)
	return false
}

const hashFileName = ".meta.csv"

func (s *scanner) scanArchive() bool {
	s.lc.Started()
	defer s.lc.Done()

	defer func() {
		s.events <- events.Progress{
			ArchivePath:   s.archivePath,
			ProgressState: events.WalkFileTreeComplete,
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
					FullName:    path,
				},
				Error: err}
			return nil
		}

		meta, err := d.Info()
		if err != nil {
			s.events <- events.ScanError{
				Meta: events.FileMeta{
					ArchivePath: s.archivePath,
					FullName:    path,
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
			FullName:    path,
			ModTime:     modTime,
			Size:        uint64(meta.Size()),
		}

		s.infos[sys.Ino] = &fileInfo{
			meta: fileMeta,
		}
		s.events <- fileMeta

		return nil
	})
	return true
}

func (s *scanner) hashArchive() bool {
	s.lc.Started()
	defer s.lc.Done()

	s.readMeta()
	defer func() {
		s.storeMeta()
	}()

	defer func() {
		s.events <- events.Progress{
			ArchivePath:   s.archivePath,
			ProgressState: events.HashFileTreeComplete,
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
				return false
			}
			s.hashFile(fileInfos[i])

			s.events <- events.FileHash{
				INode:       info.meta.INode,
				ArchivePath: info.meta.ArchivePath,
				Hash:        info.hash,
			}
		}
	}
	return true
}

func (s *scanner) hashFile(info *fileInfo) {
	hash := sha256.New()
	buf := make([]byte, 1024*1024)

	fsys := os.DirFS(s.archivePath)
	file, err := fsys.Open(info.meta.FullName)
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
		s.events <- events.Progress{
			ArchivePath:   s.archivePath,
			ProgressState: events.HashFileTree,
			Processed:     s.totalHashed,
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
				s.events <- events.Progress{
					ArchivePath:   s.archivePath,
					ProgressState: events.HashFileTree,
					Processed:     s.totalHashed,
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
			norm.NFC.String(info.meta.FullName),
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

func (s *scanner) copy(from events.FileMeta) bool {
	log.Printf("### copy from %s to %s", from.AbsName(), s.archivePath)
	return true
}

func (s *scanner) move(oldMeta, newMeta events.FileMeta) bool {
	log.Printf("### rename from %#v as %#v", oldMeta.AbsName(), newMeta.AbsName())
	return true
}

func (s *scanner) remove(file events.FileMeta) bool {
	log.Printf("### remove file %#v", file.AbsName())
	return true
}
