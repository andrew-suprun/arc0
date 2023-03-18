package mock_fs

import (
	"arch/files"
	"log"
	"math/rand"
	"time"

	"github.com/go-faker/faker/v4"
)

type mockFs struct{}

func NewFs() files.FS {
	return &mockFs{}
}

func (fs *mockFs) IsValid(path string) bool {
	return true
}

type file struct {
	folder string
	name   string
	size   int
}

func (fsys *mockFs) Scan(path string) <-chan any {
	log.Println("mock scan", path)
	result := make(chan any)
	go func() {
		folder := faker.Sentence()
		scanFiles := make([]file, rand.Int()%100+20)
		total_size := 0
		for i := range scanFiles {
			if i%10 == 0 {
				folder = faker.Sentence()
			}
			scanFiles[i].name = faker.Sentence()
			scanFiles[i].size = rand.Int() % 1000000
			scanFiles[i].folder = folder
			total_size += scanFiles[i].size
		}

		total_hashed := 0
		for _, file := range scanFiles {
			hashed := 0
			for hashed < file.size {
				if total_hashed > total_size {
					total_hashed = total_size
				}
				result <- files.ScanState{
					Archive:     path,
					Folder:      file.folder,
					Name:        file.name,
					Size:        file.size,
					Hashed:      hashed,
					TotalSize:   total_size,
					TotalToHash: total_size,
					TotalHashed: total_hashed,
				}
				hashed += 1000
				total_hashed += 1000
				time.Sleep(100 * time.Microsecond)
			}
		}
		result <- &files.ArchiveInfo{
			Archive: path,
		}
		close(result)
	}()
	return result
}

func (fs *mockFs) Stop() {
}
