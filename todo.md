* build tree: fix FileInfo.Name vs. FileMeta.FullName
* fix mock FSs

* remove directories that only contain .DS_Store and ._* files

* scanner:

func copyFile(src, dst string) error {
	err := copyFileInternal(src, dst)
	if err != nil {
		return err
	}
	return setFileModTime(src, dst)
}

