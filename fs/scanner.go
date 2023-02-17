package fs

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"scanner/lifecycle"
	"scanner/meta"

	"golang.org/x/text/unicode/norm"
)

type ScanFileResult *meta.FileMeta

type ScanStat struct {
	Base        string
	Path        string
	Size        int
	Hashed      int
	TotalSize   int
	TotalToHash int
	TotalHashed int
}

type ScanError struct {
	Path  string
	Error error
}

func Scan(lc *lifecycle.Lifecycle, base string, results chan any) {
	lc.Started()
	defer lc.Done()

	path, err := filepath.Abs(base)
	path = norm.NFC.String(path)
	if err != nil {
		results <- ScanError{Path: path, Error: err}
		return
	}

	fsys := os.DirFS(base)

	infos := collectMeta(lc, fsys, results)

	defer meta.StoreMeta(path, infos)

	inodes := map[uint64]*meta.FileMeta{}
	for _, meta := range infos {
		inodes[meta.Ino] = meta
	}

	storedMetas := meta.ReadMeta(path)

	for _, storedInfo := range storedMetas {
		if info, ok := inodes[storedInfo.Ino]; ok {
			if storedInfo.Size == info.Size {
				info.Hash = storedInfo.Hash
			}
		} else {
			fmt.Println("not found", storedInfo.Path)
		}
	}

	var (
		totalSize       int
		totalSizeToHash int
		totalHashed     int
		hash            = sha256.New()
		buf             = make([]byte, 4*1024*1024)
	)

	for _, info := range infos {
		totalSize += info.Size
		if info.Hash == "" {
			totalSizeToHash += info.Size
		}
	}

	hashFile := func(info *meta.FileMeta) {
		hash.Reset()

		file, err := fsys.Open(info.Path)
		if err != nil {
			results <- ScanError{Path: info.Path, Error: err}
			return
		}
		defer file.Close()

		var hashed int

		for {
			if lc.ShoudStop() {
				return
			}

			nr, er := file.Read(buf)
			if nr > 0 {
				nw, ew := hash.Write(buf[0:nr])
				if ew != nil {
					if err != nil {
						results <- ScanError{Path: info.Path, Error: err}
						return
					}
				}
				if nr != nw {
					results <- ScanError{Path: info.Path, Error: io.ErrShortWrite}
					return
				}
			}

			hashed += nr
			totalHashed += nr

			if er == io.EOF {
				break
			}
			if er != nil {
				results <- ScanError{Path: info.Path, Error: err}
				return
			}

			results <- ScanStat{
				Base:        base,
				Path:        info.Path,
				Size:        info.Size,
				Hashed:      hashed,
				TotalSize:   totalSize,
				TotalToHash: totalSizeToHash,
				TotalHashed: totalHashed,
			}
		}
		info.Hash = base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
		results <- ScanFileResult(info)
	}

	for _, info := range infos {
		if info.Hash == "" {
			if lc.ShoudStop() {
				return
			}
			hashFile(info)
		}
	}
}

func collectMeta(lc *lifecycle.Lifecycle, fsys fs.FS, results chan<- any) (metas meta.FileMetas) {
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			results <- ScanError{Path: path, Error: err}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			results <- ScanError{Path: path, Error: err}
			return nil
		}
		sys := info.Sys().(*syscall.Stat_t)
		metas = append(metas, &meta.FileMeta{
			Ino:     sys.Ino,
			Path:    path,
			Size:    int(sys.Size),
			ModTime: info.ModTime(),
		})
		return nil
	})
	sort.Slice(metas, func(i, j int) bool {
		return strings.ToLower(metas[i].Path) < strings.ToLower(metas[j].Path)
	})
	return metas
}
