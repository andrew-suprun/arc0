package fs

import (
	"arch/msg"
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
	"syscall"
	"time"

	"golang.org/x/text/unicode/norm"
)

func (r *runner) scan(base string) {
	path, err := filepath.Abs(base)
	path = norm.NFC.String(path)
	if err != nil {
		r.out <- msg.ScanError{Base: base, Path: path, Error: err}
		return
	}

	metas := r.collectMeta(base)
	defer func() {
		storeMeta(path, metas)
		r.out <- msg.ScanMetas{Base: base, Metas: metas}
	}()

	inodes := map[uint64]*msg.FileMeta{}
	for _, meta := range metas {
		inodes[meta.Ino] = meta
	}

	storedMetas := readMeta(path)

	for _, storedInfo := range storedMetas {
		if meta, ok := inodes[storedInfo.Ino]; ok {
			if storedInfo.Size == meta.Size {
				meta.Hash = storedInfo.Hash
			}
		} else {
			log.Println("not found", storedInfo.Path)
		}
	}

	var (
		totalSize       int
		totalSizeToHash int
		totalHashed     int
		hash            = sha256.New()
		buf             = make([]byte, 4*1024*1024)
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

	hashFile := func(meta *msg.FileMeta) {
		defer func() {
			totalHashed += meta.Size
		}()

		hash.Reset()

		file, err := os.Open(meta.Path)
		if err != nil {
			r.out <- msg.ScanError{Base: base, Path: meta.Path, Error: err}
			return
		}
		defer file.Close()

		var hashed int

		for {
			if r.ShoudStop() {
				return
			}

			nr, er := file.Read(buf)
			if nr > 0 {
				nw, ew := hash.Write(buf[0:nr])
				if ew != nil {
					if err != nil {
						r.out <- msg.ScanError{Base: base, Path: meta.Path, Error: err}
						return
					}
				}
				if nr != nw {
					r.out <- msg.ScanError{Base: base, Path: meta.Path, Error: io.ErrShortWrite}
					return
				}
			}

			hashed += nr

			if er == io.EOF {
				break
			}
			if er != nil {
				r.out <- msg.ScanError{Base: base, Path: meta.Path, Error: err}
				return
			}

			r.out <- msg.ScanStat{
				Base:        base,
				Path:        meta.Path,
				Size:        meta.Size,
				Hashed:      hashed,
				TotalSize:   totalSize,
				TotalToHash: totalSizeToHash,
				TotalHashed: totalHashed + hashed,
			}
		}
		meta.Hash = base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
	}

	for _, meta := range metas {
		if meta.Hash == "" {
			if r.ShoudStop() {
				return
			}
			hashFile(meta)
		}
	}
}

func (r *runner) collectMeta(base string) (metas msg.FileMetas) {
	filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if r.ShoudStop() || !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		if err != nil {
			r.out <- msg.ScanError{Base: base, Path: path, Error: err}
			return nil
		}

		meta, err := d.Info()
		if err != nil {
			r.out <- msg.ScanError{Base: base, Path: path, Error: err}
			return nil
		}
		sys := meta.Sys().(*syscall.Stat_t)
		metas = append(metas, &msg.FileMeta{
			Ino:     sys.Ino,
			Path:    path,
			Size:    int(sys.Size),
			ModTime: meta.ModTime(),
		})
		return nil
	})
	sort.Slice(metas, func(i, j int) bool {
		return strings.ToLower(metas[i].Path) < strings.ToLower(metas[j].Path)
	})
	return metas
}

const HashFileName = ".meta.csv"

func readMeta(basePath string) (result msg.FileMetas) {
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
		if len(record) < 5 {
			continue
		}
		ino, er1 := strconv.ParseUint(record[0], 10, 64)
		size, er2 := strconv.ParseInt(record[2], 10, 64)
		modTime, er3 := time.Parse(time.RFC3339, record[3])
		if er1 != nil || er2 != nil || er3 != nil {
			continue
		}
		result = append(result, &msg.FileMeta{
			Ino:     ino,
			Path:    norm.NFC.String(record[1]),
			Size:    int(size),
			ModTime: modTime,
			Hash:    record[4],
		})
	}

	return result
}

func storeMeta(basePath string, metas msg.FileMetas) error {
	result := make([][]string, len(metas)+1)
	result[0] = []string{"Inode", "Path", "Size", "ModTime", "Hash"}

	for i, meta := range metas {
		result[i+1] = []string{
			fmt.Sprint(meta.Ino),
			norm.NFC.String(meta.Path),
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
