* restore scan progress indicatore
* ??? move Screen{} and View() into separate package
* remove empty folders
* file copied/deleted/moved events
* file operations
* Separate Scroll into Scroll and Sized
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

