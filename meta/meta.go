package meta

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type FileMeta struct {
	Ino     uint64
	Path    string
	Size    int64
	ModTime time.Time
	Hash    string
}

type FileMetas = []*FileMeta

const HashFileName = ".meta.csv"

func ReadMeta(basePath string) (result []*FileMeta) {
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
		result = append(result, &FileMeta{
			Ino:     ino,
			Path:    record[1],
			Size:    size,
			ModTime: modTime,
			Hash:    record[4],
		})
	}

	return result
}

func StoreMeta(basePath string, metas []*FileMeta) error {
	result := make([][]string, len(metas)+1)
	result[0] = []string{"Inode", "Path", "Size", "ModTime", "Hash"}

	for i, meta := range metas {
		result[i+1] = []string{
			fmt.Sprint(meta.Ino),
			meta.Path,
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
