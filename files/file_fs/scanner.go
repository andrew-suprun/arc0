package file_fs

import (
	"arch/lifecycle"
	m "arch/model"
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
	root   m.Root
	events m.EventChan
	lc     *lifecycle.Lifecycle
	infos  map[uint64]*fileInfo
}

type fileInfo struct {
	meta m.FileMeta
	hash m.Hash
}

const hashFileName = ".meta.csv"

func (s *scanner) Send(cmd m.FileCommand) {
	switch cmd := cmd.(type) {
	case m.ScanArchive:
		s.scanArchive()

	case m.HashArchive:
		s.hashArchive()

	case m.HandleFiles:
		_ = cmd
		// TODO
	}
}

func (s *scanner) scanArchive() {
	fsys := os.DirFS(s.root.String())
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if s.lc.ShoudStop() || !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		if err != nil {
			s.events <- m.Error{
				Name:  m.Name{Path: m.Path(dir(path)), Base: m.Base(name(path))},
				Error: err}
			return nil
		}

		meta, err := d.Info()
		if err != nil {
			s.events <- m.Error{
				Name:  m.Name{Path: m.Path(dir(path)), Base: m.Base(name(path))},
				Error: err}
			return nil
		}
		sys := meta.Sys().(*syscall.Stat_t)
		modTime := meta.ModTime()
		modTime = modTime.UTC().Round(time.Second)

		fileMeta := m.FileMeta{
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

		s.infos[sys.Ino] = &fileInfo{
			meta: fileMeta,
		}

		return nil
	})

	result := m.ArchiveScanned{Root: s.root}
	for _, info := range s.infos {
		result.FileMetas = append(result.FileMetas, info.meta)
	}
	s.events <- result
}

func (s *scanner) hashArchive() {
	s.lc.Started()
	defer s.lc.Done()

	s.readMeta()
	defer func() {
		s.storeMeta()
	}()

	defer func() {
		s.events <- m.Progress{
			Root:          s.root,
			ProgressState: m.FileTreeHashed,
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

			s.events <- m.FileHashed{
				Id:   info.meta.Id,
				Hash: info.hash,
			}
		}
	}
}

func (s *scanner) hashFile(info *fileInfo) {
	hash := sha256.New()
	buf := make([]byte, 1024*1024)
	var hashed uint64

	fsys := os.DirFS(info.meta.Root.String())
	file, err := fsys.Open(info.meta.Name.String())
	if err != nil {
		s.events <- m.Error{Name: info.meta.Name, Error: err}
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
					s.events <- m.Error{Name: info.meta.Name, Error: err}
					return
				}
			}
			if nr != nw {
				s.events <- m.Error{Name: info.meta.Name, Error: err}
				return
			}
		}

		if er == io.EOF {
			break
		}
		if er != nil {
			s.events <- m.Error{Name: info.meta.Name, Error: er}
			return
		}

		hashed += uint64(nr)
		s.events <- m.Progress{
			Root:          info.meta.Root,
			ProgressState: m.HashingFile,
			HandledSize:   hashed,
		}
	}
	info.hash = m.Hash(base64.RawURLEncoding.EncodeToString(hash.Sum(nil)))
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

			info, ok := s.infos[iNode]
			if hash != "" && ok && info.meta.ModTime == modTime && info.meta.Size == size {
				info.hash = m.Hash(hash)
				s.events <- m.FileHashed{
					Id: m.Id{
						Root: s.root,
						Name: m.Name{
							Path: info.meta.Path,
							Base: info.meta.Base,
						},
					},
					Hash: m.Hash(hash),
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
			norm.NFC.String(info.meta.Name.String()),
			fmt.Sprint(info.meta.Size),
			info.meta.ModTime.UTC().Format(time.RFC3339Nano),
			info.hash.String(),
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
