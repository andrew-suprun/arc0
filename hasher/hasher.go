package hasher

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"io/fs"
)

type FileHashResult struct {
	Path string
	Hash string
}

type FileHashStat struct {
	Path   string
	Hashed int64
}

type FileHashError struct {
	Path  string
	Error error
}

func HashFile(ctx context.Context, fsys fs.FS, path string) (results chan any) {
	var (
		file fs.File
		hash = sha256.New()
		buf  = make([]byte, 16*1024*1024)
		err  error
	)

	results = make(chan any)

	hashFile := func() {
		defer close(results)

		file, err = fsys.Open(path)
		if err != nil {
			results <- FileHashError{Path: path, Error: err}
			return
		}
		defer file.Close()

		hash.Reset()
		var hashed int64

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
						results <- FileHashError{Path: path, Error: err}
						return
					}
				}
				if nr != nw {
					results <- FileHashError{Path: path, Error: io.ErrShortWrite}
					return
				}
				hashed += int64(nr)
			}
			if er == io.EOF {
				break
			}
			if er != nil {
				results <- FileHashError{Path: path, Error: err}
				return
			}

			results <- FileHashStat{Path: path, Hashed: hashed}
		}
		results <- FileHashResult{Path: path, Hash: base64.RawURLEncoding.EncodeToString(hash.Sum(nil))}
	}

	go hashFile()
	return results
}
