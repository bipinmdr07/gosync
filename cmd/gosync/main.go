package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

// Define CLI flags
var source = flag.String("source", "", "The path to source directory. (Required)")
var destination = flag.String("dest", "", "The path to destination directory. (Required)")
var doDelete = flag.Bool("delete", false, "If present delete extra files and folders from destination.")
var dryRun = flag.Bool("dryRun", false, "If present what operation are performed without changing anything.")
var verbose = flag.Bool("verbose", false, "If present enable detailed logging of operation.")
var workers = flag.Int("workers", runtime.NumCPU(), "Specifices the number of concurrent file copy workers.")

func main() {
	// Parse flags
	flag.Parse()

	// Validate mandatory fields.
	if *source == "" || *destination == "" {
		fmt.Fprintf(os.Stderr, "Error: --source and --dest are required arguments. \n")
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("Source: %v, Dest: %v, doDelete: %v, DryRun: %v, Verbose: %v, Workers: %d\n", *source, *destination, *doDelete, *dryRun, *verbose, *workers)
}
