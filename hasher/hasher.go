package hasher

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

type FS interface {
	Open(name string) (File, error)
}

type File interface {
	Read([]byte) (int, error)
	Close() error
}

type FileHash struct {
	Path string
	Hash string
	Read int
	Err  error
}

func Run(ctx context.Context, fs FS, paths <-chan string, results chan<- FileHash) {
	var (
		file File
		read int
		path string
		hash = sha256.New()
		buf  = make([]byte, 1024*1024)
		err  error
		done bool
	)

	hashFile := func() {
		file, err = fs.Open(path)
		if err != nil {
			results <- FileHash{Path: path, Err: err}
			return
		}
		defer file.Close()

		hash.Reset()
		read = 0

		for {
			select {
			case <-ctx.Done():
				done = true
				return
			default:
			}

			nr, er := file.Read(buf)
			if nr > 0 {
				nw, ew := hash.Write(buf[0:nr])
				if ew != nil {
					if err != nil {
						results <- FileHash{Path: path, Err: err}
						return
					}
				}
				if nr != nw {
					results <- FileHash{Path: path, Err: io.ErrShortWrite}
					return
				}
				read += nr
			}
			if er == io.EOF {
				break
			}
			if er != nil {
				results <- FileHash{Path: path, Err: err}
				return
			}
			results <- FileHash{Path: path, Read: read}
		}
		results <- FileHash{Path: path, Read: read, Hash: base64.RawURLEncoding.EncodeToString(hash.Sum(nil))}
	}

	for !done {
		select {
		case path = <-paths:
			hashFile()
		case <-ctx.Done():
			return
		}
	}
	fmt.Println("Done")
}
