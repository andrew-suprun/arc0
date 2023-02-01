package fs

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"scanner/meta"
)

type ScanFileResult *meta.FileMeta

type ScanStat struct {
	Path        string
	Size        int
	Hashed      int
	TotalToHash int
	TotalHashed int
}

type ScanError struct {
	Path  string
	Error error
}

func Scan(ctx context.Context, base string) (results chan any) {
	results = make(chan any)

	go func() {
		path, err := filepath.Abs(base)
		if err != nil {
			results <- ScanError{Path: path, Error: err}
			return
		}

		fsys := os.DirFS(base)
		defer close(results)

		infos := collectMeta(ctx, fsys, results)
		defer meta.StoreMeta(path, infos)

		inodes := map[uint64]*meta.FileMeta{}
		for _, meta := range infos {
			inodes[meta.Ino] = meta
		}

		storedMetas := meta.ReadMeta(path)

		for _, storedInfo := range storedMetas {
			if info, ok := inodes[storedInfo.Ino]; ok {
				if storedInfo.ModTime.UTC() == info.ModTime.UTC() && storedInfo.Size == info.Size {
					info.Hash = storedInfo.Hash
				}
			}
		}

		var (
			totalSizeToHash int
			totalHashed     int
			hash            = sha256.New()
			buf             = make([]byte, 4*1024*1024)
		)

		for _, info := range infos {
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
				select {
				case <-ctx.Done():
					return
				default:
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
					Path:        info.Path,
					Size:        info.Size,
					Hashed:      hashed,
					TotalToHash: totalSizeToHash,
					TotalHashed: totalHashed,
				}
			}
			info.Hash = base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
			results <- ScanFileResult(info)
		}

		for _, info := range infos {
			if info.Hash == "" {
				select {
				case <-ctx.Done():
					return
				default:
				}
				hashFile(info)
			}
		}
	}()

	return results
}

func collectMeta(ctx context.Context, fsys fs.FS, results chan<- any) (metas meta.FileMetas) {
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			results <- ScanError{Path: path, Error: err}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if d.Name() == meta.HashFileName || d.Name() == ".DS_Store" {
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
		return metas[i].Path < metas[j].Path
	})
	return metas
}
