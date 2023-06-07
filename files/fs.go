package files

type FS interface {
	NewScanner(archivePath string) Scanner
}

type Scanner interface {
	Handler(msg any)
}

type ScanArchive struct{}
type HashArchive struct{}
