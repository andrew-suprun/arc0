package meta

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

type FileMeta struct {
	Path    string
	Size    int64
	ModTime time.Time
	Hash    string
}

const HashFileName = ".meta.json"

func ReadMeta(basePath string) (result map[uint64]*FileMeta) {
	absHashFileName := filepath.Join(basePath, HashFileName)

	hashInfoFile, err := os.Open(absHashFileName)
	if err == nil {
		defer hashInfoFile.Close()

		buf, err := io.ReadAll(hashInfoFile)
		if err != nil {
			return result
		}
		json.Unmarshal(buf, &result)
	}
	return result
}

func StoreMeta(basePath string, metas map[uint64]*FileMeta) error {
	absHashFileName := filepath.Join(basePath, HashFileName)

	buf, err := json.MarshalIndent(metas, "", "    ")
	if err != nil {
		return err
	}

	hashFile, err := os.Create(absHashFileName)
	if err != nil {
		return err
	}
	defer hashFile.Close()

	_, err = hashFile.Write(buf)
	if err != nil {
		return err
	}
	return nil
}
