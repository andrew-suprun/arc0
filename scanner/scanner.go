package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"

	"scanner/hasher"
	"scanner/meta"
)

type ScanFileResult *meta.FileMeta
type ScanResult meta.FileMetas

type ScanStat struct {
	Path        string
	Size        int64
	Hashed      int64
	TotalToHash int64
	TotalHashed int64
}

type ScanError struct {
	Path  string
	Error error
}

const threads = 1

func Scan(ctx context.Context, base string) (results chan any) {
	fmt.Println("scan", base)
	results = make(chan any)

	path, err := filepath.Abs(base)
	if err != nil {
		results <- ScanError{Path: path, Error: err}
		return
	}

	fsys := os.DirFS(base)

	wg := sync.WaitGroup{}
	wg.Add(threads)

	go func() {
		defer close(results)

		infos := collectMeta(ctx, fsys, results)
		paths := map[string]*meta.FileMeta{}
		inodes := map[uint64]*meta.FileMeta{}
		for _, meta := range infos {
			paths[meta.Path] = meta
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

		var totalSizeToHash int64
		for _, info := range infos {
			if info.Hash == "" {
				totalSizeToHash += info.Size
			}
		}

		var totalHashed atomic.Int64
		requests := make(chan string)

		for i := 0; i < threads; i++ {
			go func() {
				for path := range requests {
					info := paths[path]
					var hashed int64
					updates := hasher.HashFile(ctx, fsys, path)
					for update := range updates {
						switch update := update.(type) {
						case hasher.FileHashStat:
							totalHashed.Add(update.Hashed - hashed)
							hashed = update.Hashed
							results <- ScanStat{
								Path:        path,
								Size:        info.Size,
								Hashed:      hashed,
								TotalToHash: totalSizeToHash,
								TotalHashed: totalHashed.Load(),
							}
						case hasher.FileHashResult:
							info.Hash = update.Hash
							totalHashed.Add(info.Size - hashed)
							results <- ScanFileResult(info)
						case hasher.FileHashError:
							totalHashed.Add(-hashed)
							results <- ScanError{
								Path:  path,
								Error: update.Error,
							}
						}
					}
				}
				wg.Done()
			}()
		}

		for _, info := range infos {
			if info.Hash == "" {
				requests <- info.Path
			}
		}
		close(requests)

		wg.Wait()

		meta.StoreMeta(path, infos)

		results <- infos
	}()

	return results
}

func collectMeta(ctx context.Context, fsys fs.FS, results chan<- any) (metas ScanResult) {
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
			Size:    sys.Size,
			ModTime: info.ModTime(),
		})
		return nil
	})
	sort.Slice(metas, func(i, j int) bool {
		if metas[i].Size > metas[j].Size {
			return true
		} else if metas[i].Size < metas[j].Size {
			return false
		}
		return metas[i].Path < metas[j].Path
	})
	return metas
}
