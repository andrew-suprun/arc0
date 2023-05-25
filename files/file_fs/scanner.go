package file_fs

import (
	"arch/model"
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
	"time"

	"syscall"

	"golang.org/x/text/unicode/norm"
)

type fileMetas map[uint64]*model.FileMeta

func (f *file_fs) scan(archivePath string) {
	f.lc.Started()
	defer f.lc.Done()

	metas := f.collectMeta(archivePath)
	defer func() {
		storeMeta(archivePath, metas)
		if f.lc.ShoudStop() {
			return
		}
		archiveFiles := make(model.FileMetas, 0, len(metas))
		for _, meta := range metas {
			archiveFiles = append(archiveFiles, meta)
		}
		f.events <- archiveFileEvent{archivePath: archivePath, archiveFiles: archiveFiles}
		f.events <- analizeArchives{}
	}()

	storedMetas := readMeta(archivePath)

	for ino, storedInfo := range storedMetas {
		if meta, ok := metas[ino]; ok {
			if storedInfo.Size == meta.Size && storedInfo.ModTime == meta.ModTime {
				meta.Hash = storedInfo.Hash
			}
		} else {
			log.Println("not found", storedInfo.FullName)
		}
	}

	var (
		totalSize       int
		totalSizeToHash int
		totalHashed     int
		hash            = sha256.New()
		buf             = make([]byte, 256*1024)
	)

	for _, meta := range metas {
		totalSize += meta.Size
		if meta.Hash == "" {
			totalSizeToHash += meta.Size
		}
	}

	if totalSizeToHash == 0 {
		return
	}

	scanStarted := time.Now()
	hashFile := func(meta *model.FileMeta) {
		defer func() {
			totalHashed += meta.Size
		}()

		hash.Reset()

		fsys := os.DirFS(archivePath)
		file, err := fsys.Open(meta.FullName)
		if err != nil {
			f.events <- scanError{archivePath: archivePath, path: meta.FullName, err: err}
			return
		}
		defer file.Close()

		var hashed int

		for {
			if f.lc.ShoudStop() {
				return
			}

			nr, er := file.Read(buf)
			if nr > 0 {
				nw, ew := hash.Write(buf[0:nr])
				if ew != nil {
					if err != nil {
						f.events <- scanError{archivePath: archivePath, path: meta.FullName, err: err}
						return
					}
				}
				if nr != nw {
					f.events <- scanError{archivePath: archivePath, path: meta.FullName, err: io.ErrShortWrite}
					return
				}
			}

			hashed += nr

			if er == io.EOF {
				break
			}
			if er != nil {
				f.events <- scanError{archivePath: archivePath, path: meta.FullName, err: err}
				return
			}

			f.events <- scanStateEvent{
				meta:      meta,
				remaining: time.Duration(float64(time.Since(scanStarted)) * float64(totalSizeToHash) / float64(totalHashed+hashed)),
				progress:  float64(totalSize-totalSizeToHash+totalHashed+hashed) / float64(totalSize)}
		}
		meta.Hash = base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
	}

	for _, meta := range metas {
		if meta.Hash == "" {
			if f.lc.ShoudStop() {
				return
			}
			hashFile(meta)
		}
	}
}

type archiveFileEvent struct {
	archivePath  string
	archiveFiles model.FileMetas
}

func (e archiveFileEvent) HandleEvent(m *model.Model) {
	for i := range m.Archives {
		if e.archivePath == m.Archives[i].Path {
			m.Archives[i].Files = e.archiveFiles
		}
	}
}

type scanStateEvent struct {
	meta      *model.FileMeta
	remaining time.Duration
	progress  float64
}

func (e scanStateEvent) HandleEvent(m *model.Model) {
	m.Archives[0].ScanState = &model.ScanState{
		Path:      filepath.Dir(e.meta.FullName),
		Name:      filepath.Base(e.meta.FullName),
		Remaining: e.remaining,
		Progress:  e.progress,
	}
}

func (f *file_fs) collectMeta(archivePath string) (infos fileMetas) {
	infos = fileMetas{}
	fsys := os.DirFS(archivePath)
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if f.lc.ShoudStop() || !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		if err != nil {
			f.events <- scanError{archivePath: archivePath, path: path, err: err}
			return nil
		}

		meta, err := d.Info()
		if err != nil {
			f.events <- scanError{archivePath: archivePath, path: path, err: err}
			return nil
		}
		sys := meta.Sys().(*syscall.Stat_t)
		modTime := meta.ModTime()
		modTime = modTime.UTC().Round(time.Second)

		infos[sys.Ino] = &model.FileMeta{
			Archive:  archivePath,
			FullName: path,
			Size:     int(sys.Size),
			ModTime:  modTime,
		}
		return nil
	})
	return infos
}

type scanError struct {
	archivePath, path string
	err               error
}

func (e scanError) HandleEvent(m *model.Model) {
	m.Errors = append(m.Errors, model.ScanError{Archive: e.archivePath, Path: e.path, Error: e.err})
}

const HashFileName = ".meta.csv"

func readMeta(basePath string) (result fileMetas) {
	result = fileMetas{}
	absHashFileName := filepath.Join(basePath, HashFileName)
	hashInfoFile, err := os.Open(absHashFileName)
	if err != nil {
		return nil
	}
	defer hashInfoFile.Close()

	records, err := csv.NewReader(hashInfoFile).ReadAll()
	if err != nil {
		return nil
	}

	for _, record := range records[1:] {
		if len(record) == 5 {
			ino, er1 := strconv.ParseUint(record[0], 10, 64)
			size, er2 := strconv.ParseInt(record[2], 10, 64)
			modTime, er3 := time.Parse(time.RFC3339, record[3])
			modTime = modTime.UTC().Round(time.Second)
			if er1 != nil || er2 != nil || er3 != nil {
				continue
			}
			result[ino] = &model.FileMeta{
				FullName: record[1],
				Size:     int(size),
				ModTime:  modTime,
				Hash:     record[4],
			}
		}
	}

	return result
}

func storeMeta(basePath string, metas fileMetas) error {
	result := make([][]string, 1, len(metas)+1)
	result[0] = []string{"Inode", "Name", "Size", "ModTime", "Hash"}

	for ino, meta := range metas {
		result = append(result, []string{
			fmt.Sprint(ino),
			norm.NFC.String(meta.FullName),
			fmt.Sprint(meta.Size),
			meta.ModTime.UTC().Format(time.RFC3339Nano),
			meta.Hash,
		})
	}

	absHashFileName := filepath.Join(basePath, HashFileName)
	hashInfoFile, err := os.Create(absHashFileName)

	if err != nil {
		return err
	}
	err = csv.NewWriter(hashInfoFile).WriteAll(result)
	hashInfoFile.Close()
	return err
}
