package mock_fs

import (
	"arch/app"
	"math/rand"
	"path/filepath"
	"time"

	"github.com/go-faker/faker/v4"
)

type mockFs struct{}

func NewFs() app.FS {
	return &mockFs{}
}

func (fs *mockFs) IsValid(path string) bool {
	return true
}

type file struct {
	name string
	size int
	hash string
}

func (fsys *mockFs) Scan(path string) <-chan any {
	result := make(chan any)
	go func() {
		scanFiles, total_size, total_hashed := genFiles()
		for i, file := range scanFiles {
			if file.hash != "" {
				continue
			}
			hashed := 0
			for hashed < file.size {
				if total_hashed > total_size {
					total_hashed = total_size
				}
				result <- app.ScanState{
					Archive:     path,
					Name:        file.name,
					Size:        file.size,
					Hashed:      hashed,
					TotalSize:   total_size,
					TotalToHash: total_size,
					TotalHashed: total_hashed,
				}
				hashed += 10000
				total_hashed += 10000
				time.Sleep(100 * time.Microsecond)
			}
			scanFiles[i].hash = faker.Phonenumber()
		}

		infos := make([]app.FileInfo, len(scanFiles))

		for i := range scanFiles {
			infos[i] = app.FileInfo{
				Archive: path,
				Name:    scanFiles[i].name,
				Size:    scanFiles[i].size,
				Hash:    scanFiles[i].hash,
			}
		}

		result <- &app.ArchiveInfo{
			Archive: path,
			Files:   infos,
		}
		close(result)
	}()
	return result
}

func (fs *mockFs) Stop() {
}

var nDirs = []int{10, 5, 3, 0}
var nFiles = []int{40, 20, 10, 5}

func genFiles() ([]file, int, int) {
	scanFiles := []file{}
	total_size := 0
	total_hashed := 0
	genFilesRec([]string{}, 0, rand.Intn(100), &scanFiles, &total_size, &total_hashed)
	return scanFiles, total_size, total_hashed
}

func genFilesRec(dirs []string, level, pcHashed int, files *[]file, total_size, total_hashed *int) {
	path := filepath.Join(dirs...)
	for i := 0; i < nFiles[level]; i++ {
		size := rand.Int() % 1000000
		*total_size += size
		hash := ""
		if rand.Intn(100) < pcHashed {
			hash = faker.Phonenumber()
			*total_hashed += size
		}
		*files = append(*files, file{name: filepath.Join(path, faker.Sentence()), size: size, hash: hash})
	}
	for i := 0; i < nDirs[level]; i++ {
		genFilesRec(append(dirs, faker.Sentence()), level+1, pcHashed, files, total_size, total_hashed)
	}
}
