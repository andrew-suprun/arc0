package scanner

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestBasic(t *testing.T) {
	fmt.Println(os.Getwd())
	fsys := os.DirFS("../../../../Downloads")
	updates := Scan(context.Background(), fsys, ".")
	for update := range updates {
		switch update := update.(type) {
		case ScanStat:
			progress := float64((update.TotalHashed+update.TotalToHash/20000)*10000/update.TotalToHash) / 100
			fmt.Printf("stat: %7.2f  file=%s size=%d hashed=%d total size=%d total hashed=%d\n", progress, update.Path, update.Size, update.Hashed, update.TotalToHash, update.TotalHashed)
		case ScanResult:
			for _, update := range update {
				fmt.Printf("hash: %12d %s %s\n", update.Size, update.Hash, update.Path)
			}
		case ScanError:
			fmt.Printf("stat: file=%s error=%#v\n", update.Path, update.Error)
		}
	}
}
