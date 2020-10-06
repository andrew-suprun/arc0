package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"time"

	"golang.org/x/text/unicode/norm"
)

var pathFlag = flag.String("path", "", "Directory to scan.")
var fromFlag = flag.String("from", "", "Directory to move files from.")
var toFlag = flag.String("to", "", "Directory to move files to.")
var dirFlag = flag.String("dir", "", "")
var fileFlag = flag.String("file", "", "")

func main() {
	log.SetFlags(0)

	signal.Notify(c, os.Interrupt)

	if len(os.Args) > 1 {
		flag.CommandLine.Parse(os.Args[2:])
		switch os.Args[1] {
		case "rehash":
			rehash()
		case "dedup":
			dedup()
		case "mirror":
			mirror()
		}
	}
}

var c = make(chan os.Signal, 1)

var gotInterrupted = false

func interrupted() bool {
	if gotInterrupted {
		return true
	}
	select {
	case s := <-c:
		log.Println("Got signal:", s)
		gotInterrupted = true
		return true
	default:
		return false
	}
}

type hashInfo struct {
	Size    int64
	Hash    string
	Mode    os.FileMode
	ModTime time.Time
}

const hashFileName = ".hashes.json"

func rehash() {
	if *pathFlag == "" || *pathFlag == "/" {
		log.Println("-path flag is required.")
		os.Exit(1)
	}
	path, err := filepath.Abs(*pathFlag)
	if err != nil {
		panic(err)
	}
	rehashPath(path)
}

type counts struct {
	count int
	name  string
}

func dedup() {
	if *pathFlag == "" || *pathFlag == "/" {
		log.Println("-path is required.")
		os.Exit(1)
	}

	basePath, err := filepath.Abs(*pathFlag)
	if err != nil {
		panic(err)
	}
	infos := rehashPath(basePath)

	hashMap := map[string][]string{}
	dirCounts := map[string]int{}

	for fileName, info := range infos {
		fileNames, ok := hashMap[info.Hash]
		if !ok {
			fileNames = []string{fileName}
		} else {
			fileNames = append(fileNames, fileName)
		}
		hashMap[info.Hash] = fileNames
	}

	if *dirFlag == "" && *fileFlag == "" {
		for hash, names := range hashMap {

			if len(names) == 1 {
				continue
			}
			fmt.Println(hash)
			sort.Strings(names)
			prevDir := ""
			for _, name := range names {
				dir := filepath.Dir(name)
				baseName := filepath.Base(name)
				dirCounts[dir]++
				if prevDir != dir {
					fmt.Println("   ", dir)
					prevDir = dir
				}
				fmt.Println("       ", baseName)
			}
		}

		countSlice := make([]counts, 0, len(dirCounts))
		for dir, count := range dirCounts {
			countSlice = append(countSlice, counts{
				count: count,
				name:  dir,
			})
		}

		sort.Slice(countSlice, func(i, j int) bool {
			return countSlice[i].count > countSlice[j].count
		})

		if len(countSlice) > 0 {
			fmt.Println("\n counts:")
			for _, dirCount := range countSlice {
				fmt.Printf("%4d %q\n", dirCount.count, dirCount.name)
			}
		}
	}

	totalRemoved := 0
	for _, names := range hashMap {
		if len(names) == 1 {
			continue
		}

		hasRemainingFile := false
		for _, name := range names {
			dir := filepath.Dir(name)
			baseName := filepath.Base(name)

			if (*dirFlag == "" || dir != *dirFlag) && (*fileFlag == "" || baseName != *fileFlag) {
				hasRemainingFile = true
				break
			}
		}

		if hasRemainingFile {
			for _, name := range names {
				dir := filepath.Dir(name)
				baseName := filepath.Base(name)

				if (*dirFlag != "" && dir == *dirFlag) || (*fileFlag != "" && baseName == *fileFlag) {
					from := filepath.Join(basePath, name)
					to := filepath.Join(basePath+".bak", name)

					fmt.Printf("moving %q\n", from)
					fmt.Printf("    to %q\n", to)
					os.MkdirAll(filepath.Dir(to), 0755)
					os.Rename(from, to)
					totalRemoved++
				}
			}
		} else {
			for _, name := range names[1:] {
				from := filepath.Join(basePath, name)
				to := filepath.Join(basePath+".bak", name)

				fmt.Printf("moving %q\n", from)
				fmt.Printf("    to %q\n", to)

				os.MkdirAll(filepath.Dir(to), 0755)
				os.Rename(from, to)
				totalRemoved++
			}
		}
	}
	if totalRemoved > 0 {
		fmt.Println("### total removed", totalRemoved)
		rehashPath(basePath)
	}
}

func mirror() {
	if *fromFlag == "" || *fromFlag == "/" {
		log.Println("-from is required.")
		os.Exit(1)
	}
	if *toFlag == "" || *toFlag == "/" {
		log.Println("-to is required.")
		os.Exit(1)
	}

	fromBase, err := filepath.Abs(*fromFlag)
	if err != nil {
		panic(err)
	}
	toBase, err := filepath.Abs(*toFlag)
	if err != nil {
		panic(err)
	}

	fromInfos := rehashPath(fromBase)
	toInfos := rehashPath(toBase)

	toOriginalInfos := map[string]hashInfo{}
	for name, info := range toInfos {
		toOriginalInfos[name] = info
	}
	originalInfosChanged := false

	for name, fromInfo := range fromInfos {
		if toInfo, ok := toInfos[name]; ok && fromInfo.Hash == toInfo.Hash {
			delete(fromInfos, name)
			delete(toInfos, name)
		}
	}

	toMap := map[string][]string{}
	for toName, toInfo := range toInfos {
		toMap[toInfo.Hash] = append(toMap[toInfo.Hash], toName)
	}

	for name, fromInfo := range fromInfos {
		if toNames, ok := toMap[fromInfo.Hash]; ok && len(toNames) > 0 {
			fmt.Printf("rename %q\n", toNames[0])
			fmt.Printf("    as %q\n", name)
			toName := filepath.Join(toBase, name)
			toDir := filepath.Dir(toName)
			os.MkdirAll(toDir, 0755)
			err = os.Rename(filepath.Join(toBase, toNames[0]), toName)
			if err != nil {
				fmt.Println("###", err)
			}
			delete(fromInfos, name)
			delete(toInfos, toNames[0])
			toMap[fromInfo.Hash] = toNames[1:]
			toOriginalInfos[name] = fromInfo
			delete(toOriginalInfos, toNames[0])
			originalInfosChanged = true
		}
	}

	for name, fromInfo := range fromInfos {
		if toNames, ok := toMap[fromInfo.Hash]; !ok || len(toNames) == 0 {
			fmt.Printf("copy   %q\n", name)
			toName := filepath.Join(toBase, name)
			toDir := filepath.Dir(toName)
			os.MkdirAll(toDir, 0755)
			err = copyFile(filepath.Join(fromBase, name), toName, fromInfo.Mode)
			if err != nil {
				fmt.Println("###", err)
			}
			delete(toInfos, name)
			toOriginalInfos[name] = fromInfo
			originalInfosChanged = true
		}
	}

	for toName := range toInfos {
		fmt.Printf("remove %q\n", toName)
		err = os.Remove(filepath.Join(toBase, toName))
		if err != nil {
			fmt.Println("###", err)
		}
		delete(toOriginalInfos, toName)
		originalInfosChanged = true
	}

	if originalInfosChanged {
		storeInfos(filepath.Join(toBase, hashFileName), toOriginalInfos)
		removeEmptyFolders(toBase)
	}
	rehashPath(toBase)
}

func infoKey(name string, size int64, modTime time.Time) string {
	return fmt.Sprintf("%s:%d:%s", filepath.Base(name), size, modTime.UTC().Format("2006-01-02:15:04:05.999999999"))
}

func rehashPath(basePath string) map[string]hashInfo {
	absHashFileName := filepath.Join(basePath, hashFileName)
	originalInfoMap := map[string]hashInfo{}
	infoMap := map[string]hashInfo{}
	newInfoMap := map[string]hashInfo{}

	hashInfoFile, err := os.Open(absHashFileName)
	if err == nil {
		buf, err := ioutil.ReadAll(hashInfoFile)
		if err != nil {
			panic(err)
		}
		hashInfoFile.Close()
		json.Unmarshal(buf, &originalInfoMap)

		for name, info := range originalInfoMap {
			infoMap[infoKey(name, info.Size, info.ModTime)] = info
		}
	}

	if err != nil && err.(*os.PathError).Err.Error() == "no such file or directory" {
		err = nil
	}
	if err != nil {
		panic(err)
	}

	shoudStore := false

	err = filepath.Walk(basePath, func(name string, info os.FileInfo, err error) error {
		name = norm.NFD.String(name)
		if err != nil || info.IsDir() || info.Size() == 0 || info.Name() == ".DS_Store" || info.Name() == hashFileName {
			return nil
		}
		relName := name[len(basePath)+1:]
		baseName := filepath.Base(name)
		key := infoKey(baseName, info.Size(), info.ModTime())

		if prevInfo, ok := infoMap[key]; ok {
			newInfoMap[relName] = prevInfo
			delete(originalInfoMap, relName)
			return nil
		}
		fmt.Printf("rehashing %q\n", relName)
		shoudStore = true

		file, err := os.Open(name)
		if err != nil {
			fmt.Printf("err.1: %v\n", err)
			return nil
		}
		defer file.Close()

		h := md5.New()
		hash, err := hashFile(h, file)
		if err != nil {
			log.Printf("FAILED to process %s: %v\n", name, err)
			return err
		}

		newInfoMap[relName] = hashInfo{
			Size:    info.Size(),
			Hash:    hash,
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
		}

		return nil
	})

	if interrupted() {
		for name, info := range originalInfoMap {
			newInfoMap[name] = info
		}
	} else {
		shoudStore = shoudStore || len(originalInfoMap) > 0
	}

	if shoudStore {
		storeInfos(absHashFileName, newInfoMap)
		removeEmptyFolders(*pathFlag)
	}

	return newInfoMap
}

func hashFile(dst hash.Hash, src *os.File) (hash string, err error) {
	written := int64(0)
	buf := make([]byte, 32*1024)
	for {
		if interrupted() {
			return "", errors.New("interrupted")
		}
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return fmt.Sprintf("%x", dst.Sum(nil)), err
}

func storeInfos(name string, infoMap map[string]hashInfo) {
	fmt.Printf("---- Storing file %q ----\n", name)

	buf, err := json.MarshalIndent(infoMap, "", "    ")
	if err != nil {
		panic(err)
	}

	hashFile, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer hashFile.Close()

	_, err = hashFile.Write(buf)
	if err != nil {
		panic(err)
	}
}

func copyFile(oldName, newName string, perm os.FileMode) error {
	from, err := os.Open(oldName)
	if err != nil {
		return err
	}
	defer from.Close()

	to, err := os.OpenFile(newName, os.O_WRONLY|os.O_CREATE, perm)
	if err != nil {
		return err
	}

	_, err = io.Copy(to, from)
	to.Close()
	if err != nil {
		return err
	}

	info, _ := os.Stat(oldName)
	return os.Chtimes(newName, time.Now(), info.ModTime())
}

func removeEmptyFolders(path string) bool {
	infos, _ := ioutil.ReadDir(path)

	empty := true
	for _, info := range infos {
		newPath := filepath.Join(path, info.Name())
		if info.IsDir() {
			emptySubfolder := removeEmptyFolders(newPath)
			empty = empty && emptySubfolder
		} else if info.Name() == ".DS_Store" {
			os.Remove(newPath)
		} else {
			empty = false
		}
	}

	if empty {
		os.Remove(path)
	}
	return empty
}
