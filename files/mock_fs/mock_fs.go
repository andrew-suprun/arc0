package mock_fs

import (
	fs "arch/files"
	"log"
	"math/rand"
	"time"

	"github.com/go-faker/faker/v4"
)

type mockFs struct{}

func NewFs() fs.FS {
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
		files := make([]file, rand.Int()%100+20)
		total_size := 0
		for i := range files {
			if i%10 == 0 {
				folder = faker.Sentence()
			}
			files[i].name = faker.Sentence()
			files[i].size = rand.Int() % 1000000
			files[i].folder = folder
			total_size += files[i].size
		}

		total_hashed := 0
		for _, file := range files {
			hashed := 0
			for hashed < file.size {
				if total_hashed > total_size {
					total_hashed = total_size
				}
				result <- fs.ScanState{
					Archive:     path,
					Folder:      file.folder,
					Name:        file.name,
					Size:        file.size,
					Hashed:      hashed,
					TotalSize:   total_size,
					TotalToHash: total_size,
					TotalHashed: total_hashed,
				}
				log.Println(path, file.folder, file.name)
				hashed += 1000
				total_hashed += 1000
				time.Sleep(100 * time.Microsecond)
			}
		}
		close(result)
	}()
	return result
}

func (fs *mockFs) Stop() {
}
