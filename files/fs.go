package files

type FS interface {
	NewScanner(archivePath string) Scanner
}

type Scanner interface {
	ScanArchive()
	HashArchive()
}
