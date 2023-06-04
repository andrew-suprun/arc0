package model

import "log"

func (m *model) analyze() {
	// m.debugPrintArchive()
}

func (m *model) debugPrintArchive() {
	log.Println("-- analyze by hash --")
	for hash, entries := range m.byHash {
		log.Printf("  hash: %q", hash)
		for i, entry := range entries {
			log.Printf("    entry: %2d a=%8q p=%8q n=%10q", i+1, entry.ArchivePath, entry.Path, entry.Name)
		}
	}

	log.Println("-- analyze by path --")
	for path, pathFolder := range m.folders {
		log.Printf("  path: %q", path)
		for i, entry := range pathFolder.entries {
			log.Printf("    entry: %2d k=%v a=%8q n=%10q h=%5q size=%d", i+1, entry.Kind, entry.ArchivePath, entry.Name, entry.Hash, entry.Size)
		}
	}
}
