package main

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"io/fs"
	"log"
	"os"
	"testing"
)

func TestFoo(t *testing.T) {
	root := "/usr/local/go/bin"
	fileSystem := os.DirFS(root)

	fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		info, err := d.Info()
		log.Println(d.Name(), info.Size())
		return err
	})
	// t.Fail()
}

var a = [1000000]byte{}

func TestFnv(t *testing.T) {
	b := base64.RawURLEncoding
	h := sha256.New224()
	h.Write([]byte(""))
	s := h.Sum(nil)
	fmt.Printf("%x\n", s)
	fmt.Println(b.EncodeToString(s))
	h = sha256.New()
	h.Write([]byte(""))
	s = h.Sum(nil)
	fmt.Printf("%x\n", s)
	fmt.Println(b.EncodeToString(s))
	h = sha512.New()
	h.Write([]byte(""))
	s = h.Sum(nil)
	fmt.Printf("%x\n", s)
	fmt.Println(b.EncodeToString(s))
	t.Fail()
}

func BenchmarkFnv64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := fnv.New64()
		h.Write(a[:])
		h.Sum64()
	}
}

func BenchmarkFnv128(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := fnv.New128()
		h.Write(a[:])
		h.Sum(nil)
	}
}

func BenchmarkMd5(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := md5.New()
		h.Write(a[:])
		h.Sum(nil)
	}
}

func BenchmarkSha256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := sha256.New()
		h.Write(a[:])
		h.Sum(nil)
	}
}

func BenchmarkSha512(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := sha512.New()
		h.Write(a[:])
		h.Sum(nil)
	}
}
