* selected -> selectedIdx, maybe?
* check gracefull shutdown in scan/hash/copy operations
* move name conflict resolution to keepFile()
* file copied/deleted/moved events
* file operations
* add descriptors to ScanErrors
* remove directories that only contain .DS_Store and ._* files
* ??? store hashes as hex encoded strings
* ??? handle 'keep all' event 

* scanner:

func copyFile(src, dst string) error {
	err := copyFileInternal(src, dst)
	if err != nil {
		return err
	}
	return setFileModTime(src, dst)
}

