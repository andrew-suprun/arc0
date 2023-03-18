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

func (r *file_fs) scan(base string, out chan any) {
	r.lc.Started()
	defer r.lc.Done()

	path, err := filepath.Abs(base)
	path = norm.NFC.String(path)
	if err != nil {
		out <- files.ScanError{Archive: base, Error: err}
		return
	}

	metas := r.collectMeta(base, out)
	defer func() {
		storeMeta(path, metas)
		out <- metas
	}()

	inodes := map[uint64]*files.FileInfo{}
	for i := range metas {
		inodes[metas[i].Ino] = &metas[i]
	}

	storedMetas := readMeta(path)

	for _, storedInfo := range storedMetas {
		if meta, ok := inodes[storedInfo.Ino]; ok {
			if storedInfo.Size == meta.Size && storedInfo.ModTime == meta.ModTime {
				meta.Hash = storedInfo.Hash
			}
		} else {
			log.Println("not found", storedInfo.Folder, storedInfo.Name)
		}
	}

	var (
		totalSize       int
		totalSizeToHash int
		totalHashed     int
		hash            = sha256.New()
		buf             = make([]byte, 16*1024)
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

	hashFile := func(meta *files.FileInfo) {
		defer func() {
			totalHashed += meta.Size
		}()

		hash.Reset()

		fsys := os.DirFS(base)
		file, err := fsys.Open(filepath.Join(meta.Folder, meta.Name))
		if err != nil {
			out <- files.ScanError{Archive: base, Folder: meta.Folder, Name: meta.Name, Error: err}
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
						out <- files.ScanError{Archive: base, Folder: meta.Folder, Name: meta.Name, Error: err}
						return
					}
				}
				if nr != nw {
					out <- files.ScanError{Archive: base, Folder: meta.Folder, Name: meta.Name, Error: io.ErrShortWrite}
					return
				}
			}

			hashed += nr

			if er == io.EOF {
				break
			}
			if er != nil {
				out <- files.ScanError{Archive: base, Folder: meta.Folder, Name: meta.Name, Error: err}
				return
			}

			out <- files.ScanState{
				Archive:     base,
				Folder:      meta.Folder,
				Name:        meta.Name,
				Size:        meta.Size,
				Hashed:      hashed,
				TotalSize:   totalSize,
				TotalToHash: totalSizeToHash,
				TotalHashed: totalHashed + hashed,
			}
		}
		meta.Hash = base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
	}

	for i := range metas {
		if metas[i].Hash == "" {
			if r.lc.ShoudStop() {
				return
			}
			hashFile(&metas[i])
		}
	}
}

func (f *file_fs) collectMeta(base string, out chan any) (infos []files.FileInfo) {
	fsys := os.DirFS(base)
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if f.lc.ShoudStop() || !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		if err != nil {
			out <- files.ScanError{Archive: base, Folder: filepath.Dir(path), Name: filepath.Base(path), Error: err}
			return nil
		}

		meta, err := d.Info()
		if err != nil {
			out <- files.ScanError{Archive: base, Folder: filepath.Dir(path), Name: filepath.Base(path), Error: err}
			return nil
		}
		sys := meta.Sys().(*syscall.Stat_t)
		modTime := meta.ModTime()
		modTime = modTime.UTC().Round(time.Second)

		infos = append(infos, files.FileInfo{
			Ino:     sys.Ino,
			Archive: base,
			Folder:  filepath.Dir(path),
			Name:    filepath.Base(path),
			Size:    int(sys.Size),
			ModTime: modTime,
		})
		return nil
	})
	return infos
}

const HashFileName = ".meta.csv"

func readMeta(basePath string) (result []files.FileInfo) {
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
		// TODO: remove after new format is stored
		if len(record) == 5 {
			ino, er1 := strconv.ParseUint(record[0], 10, 64)
			size, er2 := strconv.ParseInt(record[2], 10, 64)
			modTime, er3 := time.Parse(time.RFC3339, record[3])
			modTime = modTime.UTC().Round(time.Second)
			if er1 != nil || er2 != nil || er3 != nil {
				continue
			}
			result = append(result, files.FileInfo{
				Ino:     ino,
				Folder:  filepath.Dir(record[1]),
				Name:    filepath.Base(record[1]),
				Size:    int(size),
				ModTime: modTime,
				Hash:    record[4],
			})
		}
		if len(record) == 6 {
			ino, er1 := strconv.ParseUint(record[0], 10, 64)
			size, er2 := strconv.ParseInt(record[3], 10, 64)
			modTime, er3 := time.Parse(time.RFC3339, record[4])
			modTime = modTime.UTC().Round(time.Second)
			if er1 != nil || er2 != nil || er3 != nil {
				continue
			}
			result = append(result, files.FileInfo{
				Ino:     ino,
				Folder:  record[1],
				Name:    record[2],
				Size:    int(size),
				ModTime: modTime,
				Hash:    record[5],
			})
		}
	}

	return result
}

func storeMeta(basePath string, metas []files.FileInfo) error {
	result := make([][]string, len(metas)+1)
	result[0] = []string{"Inode", "Folder", "Name", "Size", "ModTime", "Hash"}

	for i, meta := range metas {
		result[i+1] = []string{
			fmt.Sprint(meta.Ino),
			norm.NFC.String(meta.Folder),
			norm.NFC.String(meta.Name),
			fmt.Sprint(meta.Size),
			meta.ModTime.UTC().Format(time.RFC3339Nano),
			meta.Hash,
		}
	}

	absHashFileName := filepath.Join(basePath, HashFileName)
	hashInfoFile, err := os.Create(absHashFileName)

	if err != nil {
		return err
	}
	defer hashInfoFile.Close()
	return csv.NewWriter(hashInfoFile).WriteAll(result)
}
