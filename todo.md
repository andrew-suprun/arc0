* Left/Right keys to switch folders
* Delete of folder marhed as Absent
* properly handle file events
* remove directories that only contain .DS_Store and ._* files
* handle 'keep all' event 
* add descriptions to ScanErrors
* ??? move Screen{} and View() into separate package
* ??? Separate Scroll into Scroll and Sized
* ??? store hashes as hex encoded strings

* scanner:

func copyFile(src, dst string) error {
	err := copyFileInternal(src, dst)
	if err != nil {
		return err
	}
	return setFileModTime(src, dst)
}

