package main

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"hash/fnv"
	"testing"
	"time"
)

func TestFnv(t *testing.T) {
	buf := [8 * 1024]byte{}
	h := fnv.New128a()
	start := time.Now()
	for offset := 0; offset <= 1024*1024*1024; offset += len(buf) {
		h.Write(buf[:])
	}
	s := time.Since(start).Seconds()
	t.Log(1024 / s)
}

func TestHello(t *testing.T) {
	buf := [8 * 1024]byte{}
	h := sha256.New()
	start := time.Now()
	for offset := 0; offset <= 1024*1024*1024; offset += len(buf) {
		h.Write(buf[:])
	}
	res := h.Sum(nil)
	s := time.Since(start).Seconds()
	t.Log(s)
	t.Log(1024 / s)
	t.Logf("%x\n", res)
	t.Log(base64.RawURLEncoding.EncodeToString(res))
}

var a = [10_000_000]byte{}

func BenchmarkFnv64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := fnv.New64a()
		h.Write(a[:])
		h.Sum64()
	}
}

func BenchmarkFnv128(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := fnv.New128a()
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
