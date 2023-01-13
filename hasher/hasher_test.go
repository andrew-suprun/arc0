package hasher

import (
	"context"
	"io"
	"testing"
)

type testFS struct{}

func (f *testFS) Open(name string) (File, error) {
	if name == "foo" {
		return &testFile{name, 10_000_000, 0}, nil
	} else {
		return &testFile{name, 5_000_000, 0}, nil
	}
}

type testFile struct {
	name string
	size int
	read int
}

func (f *testFile) Read(buf []byte) (int, error) {
	if f.read >= f.size {
		return 0, io.EOF
	}

	toRead := len(buf)
	if toRead > f.size-f.read {
		toRead = f.size - f.read
	}

	for i := 0; i < toRead; i++ {
		buf[i] = f.name[0]
	}
	f.read += toRead

	return toRead, nil
}

func (f *testFile) Close() error {
	return nil
}

func TestHash(t *testing.T) {
	fs := &testFS{}
	paths := make(chan string)
	results := make(chan FileHash)

	go Run(context.Background(), fs, paths, results)
	go Run(context.Background(), fs, paths, results)
	paths <- "foo"
	paths <- "bar"
	nStats := 0
	nFiles := 0
	for {
		result := <-results
		nStats++
		if result.Hash != "" {
			if result.Path == "bar" && (result.Hash != "xg_laQDWK4gJy_S58Xy1Mi-5hJhL2Ia0E74jdXkdCpY" || result.Read != 5_000_000) {
				t.Fail()
			}
			if result.Path == "foo" && (result.Hash != "ZNbj3EoorUb-FI28kgSoFktKCvYzErHTBR0wpIlXYLg" || result.Read != 10_000_000) {
				t.Fail()
			}
			nFiles++
			if nFiles == 2 {
				break
			}
		}
	}
	if nStats != 17 {
		t.Fail()
	}
}
