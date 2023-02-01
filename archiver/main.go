package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"scanner/fs"
)

var pathFlag = flag.String("path", "", "Directory to scan.")

func main() {
	log.SetFlags(log.Lshortfile)

	signal.Notify(c, os.Interrupt)

	if len(os.Args) > 1 {
		flag.CommandLine.Parse(os.Args[2:])
		switch os.Args[1] {
		case "hash":
			hash()
			// case "dedup":
			// 	dedup()
			// case "mirror":
			// 	mirror()
			// case "merge":
			// 	merge()
		}
	}
}

func hash() {
	if *pathFlag == "" || *pathFlag == "/" {
		log.Println("-path flag is required.")
		os.Exit(1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	results := fs.Scan(ctx, *pathFlag)
	start := time.Now()
	for result := range results {
		if interrupted() {
			cancel()
		}
		switch update := result.(type) {
		case fs.ScanFileResult:
			fmt.Printf("%12d %s\n", update.Size, update.Path)
		case fs.ScanStat:
			progress := float64(update.TotalHashed) / float64(update.TotalToHash)
			dur := time.Since(start)
			speed := float64(update.TotalHashed/(1024*1024)) / dur.Seconds()
			eta := start.Add(time.Duration(float64(dur) / progress))
			fmt.Printf("\033[G%6.2f%% %5.1fMiB/s %-8v ETA: %v",
				progress*100, speed, time.Until(eta).Truncate(time.Second), eta.Format(time.Stamp))

		case fs.ScanError:
			log.Printf("stat: file=%s error=%#v, %#v\n", update.Path, update.Error, errors.Unwrap(update.Error))
		}
	}
	fmt.Println()
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
