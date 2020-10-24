package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"
)

var pathFlag = flag.String("path", "", "Directory to scan.")
var fromFlag = flag.String("from", "", "Directory to move files from.")
var toFlag = flag.String("to", "", "Directory to move files to.")
var dirFlag = flag.String("dir", "", "")

const extrasDir = "~~~extras~~~"
const dupsDir = "~~~dups~~~"
const hashFileName = ".hashes.json"

func main() {
	log.SetFlags(0)

	signal.Notify(c, os.Interrupt)

	if len(os.Args) > 1 {
		flag.CommandLine.Parse(os.Args[2:])
		switch os.Args[1] {
		case "hash":
			hash()
		case "dedup":
			dedup()
		case "mirror":
			mirror()
		case "merge":
			merge()
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
	ModTime time.Time
}

func hash() {
	if *pathFlag == "" || *pathFlag == "/" {
		log.Println("-path flag is required.")
		os.Exit(1)
	}
	path, err := filepath.Abs(*pathFlag)
	if err != nil {
		panic(err)
	}
	hashPath(path)
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
	infos := hashPath(basePath)

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

	if *dirFlag == "" {
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
			if filepath.Dir(name) == *dirFlag {
				hasRemainingFile = true
				break
			}
		}

		if hasRemainingFile {
			for _, name := range names {
				if filepath.Dir(name) != *dirFlag {
					from := filepath.Join(basePath, name)
					to := filepath.Join(basePath, dupsDir, name)

					fmt.Printf("moving %q\n", from)
					fmt.Printf("    to %q\n", to)
					os.MkdirAll(filepath.Dir(to), 0755)
					os.Rename(from, to)
					totalRemoved++
				}
			}
		}
	}
	if totalRemoved > 0 {
		fmt.Println("### total removed", totalRemoved)
		hashPath(basePath)
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

	fromInfos := hashPath(fromBase)
	if interrupted() {
		return
	}
	toInfos := hashPath(toBase)
	if interrupted() {
		return
	}

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

	// TODO: unnecessary?
	// fromMap := map[string][]string{}
	// for fromName, fromInfo := range fromInfos {
	// 	fromMap[fromInfo.Hash] = append(fromMap[fromInfo.Hash], fromName)
	// }

	// for toName, toInfo := range toInfos {
	// 	if fromNames, ok := fromMap[toInfo.Hash]; ok && len(fromNames) > 0 {
	// 		fromName := fromNames[0]
	// 		from := filepath.Join(toBase, toName)
	// 		to := filepath.Join(toBase, fromName)

	// 		fmt.Printf("moving %q\n", from)
	// 		fmt.Printf("    to %q\n", to)

	// 		os.MkdirAll(filepath.Dir(to), 0755)
	// 		err = os.Rename(from, to)
	// 		if err != nil {
	// 			fmt.Println("###:1", err)
	// 		}

	// 		fromMap[toInfo.Hash] = fromNames[1:]
	// 		delete(fromInfos, fromName)
	// 		delete(toInfos, toName)
	// 		delete(toOriginalInfos, toName)
	// 		toOriginalInfos[fromName] = toInfo
	// 		originalInfosChanged = true
	// 	}
	// }

	toMap := map[string][]string{}
	for toName, toInfo := range toInfos {
		toMap[toInfo.Hash] = append(toMap[toInfo.Hash], toName)
	}

	for name, fromInfo := range fromInfos {
		if interrupted() {
			break
		}
		if toNames, ok := toMap[fromInfo.Hash]; ok && len(toNames) > 0 {
			fmt.Printf("rename %q\n", toNames[0])
			fmt.Printf("    as %q\n", name)
			toName := filepath.Join(toBase, name)
			toDir := filepath.Dir(toName)
			os.MkdirAll(toDir, 0755)
			err = os.Rename(filepath.Join(toBase, toNames[0]), toName)
			if err != nil {
				fmt.Println("###:2", err)
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
		if interrupted() {
			break
		}
		if toNames, ok := toMap[fromInfo.Hash]; !ok || len(toNames) == 0 {
			fmt.Printf("copy   %q\n", name)
			toName := filepath.Join(toBase, name)
			toDir := filepath.Dir(toName)
			os.MkdirAll(toDir, 0755)
			err = copyFile(filepath.Join(fromBase, name), toName)
			if err != nil {
				fmt.Println("###:3", err)
			}
			delete(toInfos, name)
			toOriginalInfos[name] = fromInfo
			originalInfosChanged = true
		}
	}

	for toName := range toInfos {
		if interrupted() {
			break
		}
		from := filepath.Join(toBase, toName)
		to := filepath.Join(toBase, extrasDir, toName)

		fmt.Printf("moving %q\n", from)
		fmt.Printf("    to %q\n", to)

		os.MkdirAll(filepath.Dir(to), 0755)
		err = os.Rename(from, to)
		if err != nil {
			fmt.Println("###:4", err)
		}

		delete(toOriginalInfos, toName)
		originalInfosChanged = true
	}

	if originalInfosChanged {
		storeInfos(filepath.Join(toBase, hashFileName), toOriginalInfos)
		removeEmptyFolders(toBase)
	}
	if interrupted() {
		return
	}
	hashPath(toBase)
}

func merge() {
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

	fromInfos := hashPath(fromBase)
	toInfos := hashPath(toBase)

	toHashes := map[string]struct{}{}
	for _, toInfo := range toInfos {
		toHashes[toInfo.Hash] = struct{}{}
	}

	toInfosChanged := false

	for fromName, fromInfo := range fromInfos {
		if _, ok := toHashes[fromInfo.Hash]; ok {
			continue
		}
		nameBase := ""
		nameExt := filepath.Ext(fromName)
		if nameExt != "" {
			nameExt = nameExt[1:]
			nameBase = fromName[:len(fromName)-len(nameExt)-1]
		} else {
			nameBase = fromName
		}

		toName := fromName
		if _, ok := toInfos[toName]; ok {
			for i := 1; ; i++ {
				toName = fmt.Sprintf("%s (%d).%s", nameBase, i, nameExt)
				if _, ok = toInfos[toName]; !ok {
					break
				}
			}
		}

		fromFileName := filepath.Join(fromBase, fromName)
		toFileName := filepath.Join(toBase, toName)

		os.MkdirAll(filepath.Dir(toFileName), 0755)
		err = os.Rename(fromFileName, toFileName)
		if err != nil {
			fmt.Printf("copy %q\n  to %q\n", fromFileName, toFileName)
			err = copyFile(fromFileName, toFileName)
		} else {
			fmt.Printf("move %q\n  to %q\n", fromFileName, toFileName)
		}

		if err != nil {
			fmt.Printf("### ERROR: %v\n", err)
		} else {
			toInfos[toName] = fromInfo
			toInfosChanged = true
		}
	}
	if toInfosChanged {
		storeInfos(filepath.Join(toBase, hashFileName), toInfos)
	}
}

func infoKey(name string, size int64, modTime time.Time) string {
	return fmt.Sprintf("%s:%d:%s", filepath.Base(name), size, modTime.UTC().Format("2006-01-02:15:04:05.999999999"))
}

func hashPath(basePath string) map[string]hashInfo {
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
		if err != nil {
			return nil
		}
		name = norm.NFC.String(name)
		if info.IsDir() && (info.Name() == extrasDir || info.Name() == dupsDir) {
			return filepath.SkipDir
		}

		if info.Name() == ".DS_Store" || strings.HasPrefix(info.Name(), "._") {
			fmt.Printf("removing %q\n", name)
			os.Remove(name)
			return nil
		}

		if info.IsDir() || info.Size() == 0 || info.Name() == hashFileName {
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
		fmt.Printf(" hashing %q\n", name)
		shoudStore = true

		hash, err := hashFile(name)
		if err != nil {
			log.Printf("FAILED to process %s: %v\n", name, err)
			return err
		}

		newInfoMap[relName] = hashInfo{
			Size:    info.Size(),
			Hash:    hash,
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

func hashFile(name string) (hash string, err error) {
	file, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()

	md5Hash := md5.New()
	buf := make([]byte, 32*1024)
	for {
		if interrupted() {
			return "", errors.New("interrupted")
		}
		nr, er := file.Read(buf)
		if nr > 0 {
			nw, ew := md5Hash.Write(buf[0:nr])
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
	return fmt.Sprintf("%x", md5Hash.Sum(nil)), err
}

func copyFile(src, dst string) error {
	err := copyFileInternal(src, dst)
	if err != nil {
		return err
	}
	return setFileModTime(src, dst)
}

func copyFileInternal(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	from, err := os.Open(src)
	if err != nil {
		return err
	}
	defer from.Close()

	os.MkdirAll(filepath.Dir(dst), 0755)
	to, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, info.Mode())
	if err != nil {
		return err
	}
	defer to.Close()

	buf := make([]byte, 32*1024)
	for {
		if interrupted() {
			return errors.New("interrupted")
		}
		nr, er := from.Read(buf)
		if nr > 0 {
			nw, ew := to.Write(buf[0:nr])
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
	return nil
}

func setFileModTime(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chtimes(dst, time.Now(), info.ModTime())
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
