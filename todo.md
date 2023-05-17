* breadcrumbs
* sort
* top and bottom lines in file view

* -----

if info.Name() == ".DS_Store" || strings.HasPrefix(info.Name(), "._") {
    log.Printf("removing %q\n", name)
    os.Remove(name)
    return nil
}

* -----

if info.Size() == 0 {
    return nil
}
