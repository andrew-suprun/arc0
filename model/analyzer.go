package model

import (
	"arch/files"
	"log"
)

func (m *model) keepOneFile(selected *File) {
	log.Printf("### keep one: selected %q %q %q", selected.ArchivePath, selected.FullName, selected.Hash)
	filesForHash := m.byHash[selected.Hash]
	byArch := map[string][]*File{}
	for _, fileForHash := range filesForHash {
		byArch[fileForHash.ArchivePath] = append(byArch[fileForHash.ArchivePath], fileForHash)
	}

	pending := false

	for _, archPath := range m.archivePaths {
		archFiles := byArch[archPath]
		log.Printf("### keep one: archPath %q, archFiles %d", archPath, len(archFiles))
		if len(archFiles) == 0 {
			m.archives[archPath].scanner.Send(files.Copy{Source: selected.FileMeta})
			pending = true
			log.Printf("### keep one: copy from %q %q to %q", selected.ArchivePath, selected.FullName, archPath)
			continue
		}
		keepIdx := 0
		for i, archFile := range archFiles {
			if archFile == selected || archFile.FullName == selected.FullName {
				keepIdx = i
				break
			}
		}
		for i, archFile := range archFiles {
			if i == keepIdx {
				if selected.FullName != archFile.FullName {
					m.archives[archPath].scanner.Send(files.Move{OldMeta: archFile.FileMeta, NewMeta: selected.FileMeta})
					pending = true
					log.Printf("### keep one: move from %q %q to %q", selected.ArchivePath, selected.FullName, archPath)
				}
			} else {
				m.archives[archPath].scanner.Send(files.Delete{File: archFile.FileMeta})
				pending = true
				log.Printf("### keep one: delete %q %q", archFile.ArchivePath, archFile.FullName)
			}
		}
	}

	if pending {
		for _, archFile := range filesForHash {
			archFile.Status = Pending
		}
	}
}
