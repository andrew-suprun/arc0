* breadcrumbs
* sort
* top and bottom lines in file view

* scanner:

if info.Name() == ".DS_Store" || strings.HasPrefix(info.Name(), "._") {
    log.Printf("removing %q\n", name)
    os.Remove(name)
    return nil
}

* scanner:

if info.Size() == 0 {
    return nil
}

* scanner 

func copyFile(src, dst string) error {
	err := copyFileInternal(src, dst)
	if err != nil {
		return err
	}
	return setFileModTime(src, dst)
}
