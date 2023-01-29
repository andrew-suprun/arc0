package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"scanner/scanner"
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
	results := scanner.Scan(ctx, *pathFlag)
	start := time.Now()
	for result := range results {
		if interrupted() {
			cancel()
		}
		switch update := result.(type) {
		case scanner.ScanFileResult:
			fmt.Printf("%12d %s\n", update.Size, update.Path)
		case scanner.ScanStat:
			progress := float64(update.TotalHashed) / float64(update.TotalToHash) * 100
			dur := time.Since(start)
			speed := float64(update.TotalHashed/(1024*1024)) / dur.Seconds()
			eta := time.Now().Add(time.Duration(float64(dur) * 100 / progress))
			fmt.Printf("\033[G%6.2f%% %7v %5.1fMiB  ETA: %v", progress, dur.Truncate(time.Second), speed, eta.Format(time.Stamp))

		case scanner.ScanResult:
			// for _, update := range update {
			// 	log.Printf("hash: %12d %s %s\n", update.Size, update.Hash, update.Path)
			// }
		case scanner.ScanError:
			log.Printf("stat: file=%s error=%v\n", update.Path, update.Error)
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
