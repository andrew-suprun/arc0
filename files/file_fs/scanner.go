package file_fs

import (
	"arch/files"
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

type fileInfos map[uint64]*files.FileInfo

func (r *file_fs) scan(archivePath string, out chan files.Event) {
	r.lc.Started()
	defer r.lc.Done()

	metas := r.collectMeta(archivePath, out)
	defer func() {
		storeMeta(archivePath, metas)
		if r.lc.ShoudStop() {
			return
		}
		fileMetas := make([]*files.FileInfo, 0, len(metas))
		for _, meta := range metas {
			fileMetas = append(fileMetas, meta)
		}
		out <- &files.ArchiveInfo{
			Archive: archivePath,
			Files:   fileMetas,
		}
		close(out)
	}()

	storedMetas := readMeta(archivePath)

	for ino, storedInfo := range storedMetas {
		if meta, ok := metas[ino]; ok {
			if storedInfo.Size == meta.Size && storedInfo.ModTime == meta.ModTime {
				meta.Hash = storedInfo.Hash
			}
		} else {
			log.Println("not found", storedInfo.Name)
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
	hashFile := func(meta *files.FileInfo) {
		defer func() {
			totalHashed += meta.Size
		}()

		hash.Reset()

		fsys := os.DirFS(archivePath)
		file, err := fsys.Open(filepath.Join(meta.Path, meta.Name))
		if err != nil {
			out <- files.ScanError{Archive: archivePath, Path: meta.Name, Error: err}
			return
		}
		defer file.Close()

		var hashed int

		for {
			if r.lc.ShoudStop() {
				return
			}

			nr, er := file.Read(buf)
			if nr > 0 {
				nw, ew := hash.Write(buf[0:nr])
				if ew != nil {
					if err != nil {
						out <- files.ScanError{Archive: archivePath, Path: meta.Name, Error: err}
						return
					}
				}
				if nr != nw {
					out <- files.ScanError{Archive: archivePath, Path: meta.Name, Error: io.ErrShortWrite}
					return
				}
			}

			hashed += nr

			if er == io.EOF {
				break
			}
			if er != nil {
				out <- files.ScanError{Archive: archivePath, Path: meta.Name, Error: err}
				return
			}

			select {
			case prevEvent := <-out:
				switch prevEvent.(type) {
				case *files.ScanState:
					// Drop previous []files.ScanState msg, if any
				default:
					out <- prevEvent
				}
			default:
			}

			// TODO: Fix eta/remainig
			dur := time.Since(scanStarted)
			remaining := time.Duration(float64(dur) * float64(totalSizeToHash) / float64(totalHashed+hashed))

			out <- &files.ScanState{
				Archive:   archivePath,
				Name:      meta.Name,
				Remaining: remaining,
				Progress:  float64(totalSize-totalSizeToHash+totalHashed+hashed) / float64(totalSize),
			}
		}
		meta.Hash = base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
	}

	for _, meta := range metas {
		if meta.Hash == "" {
			if r.lc.ShoudStop() {
				return
			}
			hashFile(meta)
		}
	}
}

func (f *file_fs) collectMeta(archive string, out chan files.Event) (infos fileInfos) {
	infos = fileInfos{}
	fsys := os.DirFS(archive)
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if f.lc.ShoudStop() || !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		if err != nil {
			out <- files.ScanError{Archive: archive, Path: path, Error: err}
			return nil
		}

		meta, err := d.Info()
		if err != nil {
			out <- files.ScanError{Archive: archive, Path: path, Error: err}
			return nil
		}
		sys := meta.Sys().(*syscall.Stat_t)
		modTime := meta.ModTime()
		modTime = modTime.UTC().Round(time.Second)

		infos[sys.Ino] = &files.FileInfo{
			Archive: archive,
			Path:    filepath.Dir(path),
			Name:    filepath.Base(path),
			Size:    int(sys.Size),
			ModTime: modTime,
		}
		return nil
	})
	return infos
}

const HashFileName = ".meta.csv"

func readMeta(basePath string) (result fileInfos) {
	result = fileInfos{}
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
			result[ino] = &files.FileInfo{
				Name:    record[1],
				Size:    int(size),
				ModTime: modTime,
				Hash:    record[4],
			}
		}
	}

	return result
}

func storeMeta(basePath string, metas fileInfos) error {
	result := make([][]string, 1, len(metas)+1)
	result[0] = []string{"Inode", "Name", "Size", "ModTime", "Hash"}

	for ino, meta := range metas {
		result = append(result, []string{
			fmt.Sprint(ino),
			norm.NFC.String(meta.Name),
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
