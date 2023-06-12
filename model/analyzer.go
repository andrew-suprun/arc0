package model

import "log"

func (m *model) keepOneFile(selected *File) {
	log.Printf("### keep one: selected %q %q %q", selected.ArchivePath, selected.FullName, selected.Hash)
	filesForHash := m.byHash[selected.Hash]
	byArch := map[string][]*File{}
	for _, fileForHash := range filesForHash {
		log.Printf("### keep one: %q %q", fileForHash.ArchivePath, fileForHash.FullName)
		byArch[selected.ArchivePath] = append(byArch[selected.ArchivePath], fileForHash)
	}
	// for arch, files := range byArch {
	// 	if len(files) == 0 {
	// 		m.archives
	// 	}
	// }
}
