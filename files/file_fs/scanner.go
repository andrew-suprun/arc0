package file_fs

import (
	"arch/lifecycle"
	"arch/model"
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
	events      model.EventChan
	lc          *lifecycle.Lifecycle
	infos       map[uint64]*fileInfo
	totalHashed uint64
}

type fileInfo struct {
	meta model.FileMeta
	hash string
}

const hashFileName = ".meta.csv"

func (s *scanner) ScanArchive(root string, events model.EventChan) {
	go s.scanArchive(root)
}

func (s *scanner) scanArchive(root string) {
	s.lc.Started()
	defer s.lc.Done()

	s.scanInfos(root)

	s.readMeta(root)
	defer func() {
		s.storeMeta(root)
	}()

	defer func() {
		s.events <- model.Progress{
			Root:          root,
			ProgressState: model.HashingFileTreeComplete,
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

			s.events <- model.FileHashed{
				Root: info.meta.Root,
				Name: info.meta.Name,
				Hash: info.hash,
			}
		}
	}
}

func (s *scanner) scanInfos(root string) {
	fsys := os.DirFS(root)
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if s.lc.ShoudStop() || !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		if err != nil {
			s.events <- model.Error{
				Meta: model.FileMeta{
					Root: root,
					Name: path,
				},
				Error: err}
			return nil
		}

		meta, err := d.Info()
		if err != nil {
			s.events <- model.Error{
				Meta: model.FileMeta{
					Root: root,
					Name: path,
				},
				Error: err}
			return nil
		}
		sys := meta.Sys().(*syscall.Stat_t)
		modTime := meta.ModTime()
		modTime = modTime.UTC().Round(time.Second)

		fileMeta := model.FileMeta{
			INode:   sys.Ino,
			Root:    root,
			Name:    path,
			ModTime: modTime,
			Size:    uint64(meta.Size()),
		}

		s.infos[sys.Ino] = &fileInfo{
			meta: fileMeta,
		}

		return nil
	})
}

func (s *scanner) hashFile(info *fileInfo) {
	hash := sha256.New()
	buf := make([]byte, 1024*1024)

	fsys := os.DirFS(info.meta.Root)
	file, err := fsys.Open(info.meta.Name)
	if err != nil {
		s.events <- model.Error{Meta: info.meta, Error: err}
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
					s.events <- model.Error{Meta: info.meta, Error: err}
					return
				}
			}
			if nr != nw {
				s.events <- model.Error{Meta: info.meta, Error: err}
				return
			}
		}

		if er == io.EOF {
			break
		}
		if er != nil {
			s.events <- model.Error{Meta: info.meta, Error: er}
			return
		}

		s.totalHashed += uint64(nr)
		s.events <- model.Progress{
			Root:          info.meta.Root,
			ProgressState: model.HashingFileTree,
			Processed:     s.totalHashed,
		}
	}
	info.hash = base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}

func (s *scanner) readMeta(root string) {
	absHashFileName := filepath.Join(root, hashFileName)
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
				s.events <- model.FileHashed{
					Root: root,
					Name: info.meta.Name,
					Hash: hash,
				}
				s.totalHashed += info.meta.Size
				s.events <- model.Progress{
					Root:          root,
					ProgressState: model.HashingFileTree,
					Processed:     s.totalHashed,
				}
			}
		}
	}
}

func (s *scanner) storeMeta(root string) error {
	result := make([][]string, 1, len(s.infos)+1)
	result[0] = []string{"INode", "Name", "Size", "ModTime", "Hash"}

	for iNode, info := range s.infos {
		result = append(result, []string{
			fmt.Sprint(iNode),
			norm.NFC.String(info.meta.Name),
			fmt.Sprint(info.meta.Size),
			info.meta.ModTime.UTC().Format(time.RFC3339Nano),
			info.hash,
		})
	}

	absHashFileName := filepath.Join(root, hashFileName)
	hashInfoFile, err := os.Create(absHashFileName)

	if err != nil {
		return err
	}
	err = csv.NewWriter(hashInfoFile).WriteAll(result)
	hashInfoFile.Close()
	return err
}
