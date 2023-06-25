package file_fs

import (
	"arch/model"
	"log"
)

func (fs *fileFs) handleFiles(msg model.HandleFiles) bool {
	log.Printf("### handleFiles: msg=%v", msg)
	return true
}
