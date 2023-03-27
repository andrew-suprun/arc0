package mock_fs

import (
	"arch/files"
	"math/rand"
	"path/filepath"
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
	name string
	size int
	hash string
}

func (fsys *mockFs) Scan(path string) <-chan any {
	result := make(chan any, 1)
	go func() {
		scanStarted := time.Now()
		scanFiles, totalSize, totalHashed := genFiles()
		toHash := totalSize - totalHashed
		newFilesHashed := 0
		for i, file := range scanFiles {
			if file.hash != "" {
				continue
			}
			fileHashed := 0
			for fileHashed < file.size {
				hashSize := 100000
				if fileHashed+hashSize > file.size {
					hashSize = file.size - fileHashed
				}
				fileHashed += hashSize
				newFilesHashed += hashSize

				if totalHashed+newFilesHashed > totalSize {
					totalHashed = totalSize - newFilesHashed
				}

				select {
				case prevEvent := <-result:
					switch prevEvent.(type) {
					case *files.ScanState:
						// Drop previous []files.ScanState msg, if any
					default:
						result <- prevEvent
					}
				default:
				}

				progress := float64(totalHashed+newFilesHashed) / float64(totalSize)
				dur := time.Since(scanStarted)
				eta := scanStarted.Add(time.Duration(float64(dur) / float64(newFilesHashed) * float64(toHash)))

				result <- &files.ScanState{
					Archive:   path,
					Name:      file.name,
					Remaining: time.Until(eta),
					Progress:  progress,
				}
				time.Sleep(time.Millisecond)
			}
			scanFiles[i].hash = faker.Phonenumber()
		}

		infos := make([]files.FileInfo, len(scanFiles))

		for i := range scanFiles {
			infos[i] = files.FileInfo{
				Archive: path,
				Name:    scanFiles[i].name,
				Size:    scanFiles[i].size,
				Hash:    scanFiles[i].hash,
			}
		}

		result <- &files.ArchiveInfo{
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
