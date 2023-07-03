* make model types stricter
* rebuild folder on every event
* ??? split view in onw package
* remove empty folders
* file copied/deleted/moved events
* file operations
* check gracefull shutdown in scan/hash/copy operations
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

