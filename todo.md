* update counters on file events
* update all files with the same hash
* file copied/deleted/moved events
* reflect stats in status line
* add descriptors to ScanErrors
* file operations
* store hashes as hex encoded strings
* remove directories that only contain .DS_Store and ._* files
* ??? handle 'keep all' event 

* scanner:

func copyFile(src, dst string) error {
	err := copyFileInternal(src, dst)
	if err != nil {
		return err
	}
	return setFileModTime(src, dst)
}

