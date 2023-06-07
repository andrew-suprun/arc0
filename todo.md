* mouse events in breadcrumbs
* analyzer
* reflect stats in status line
* add descriptors to ScanErrors
* conflict resolution
* file operations
* store hashes as hex encoded strings
* remove directories that only contain .DS_Store and ._* files

* scanner:

func copyFile(src, dst string) error {
	err := copyFileInternal(src, dst)
	if err != nil {
		return err
	}
	return setFileModTime(src, dst)
}

