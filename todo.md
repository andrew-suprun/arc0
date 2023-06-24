* sort by name within controller.byHash slices (for tab jumping)
* update statuses and counters on all files on file events
* populate view folders on fileHashed event
* refactor status per hash
* change meta.counts to status ('Duplicated', 'Missing')
* remove INode from FileMeta
* eliminate ScanArchive/FileScanned
* rename 'renderers' package to 'renderer'
* move 'Renderer' interface to 'renderer' package
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

