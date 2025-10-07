package cmd

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"gosync/pkg/syncer"

	"github.com/spf13/cobra"
)

var opts = &syncer.SyncOptions{}

var rootCmd = &cobra.Command{
	Use:   "gosync",
	Short: "One-way directory synchronization utility",
	Long: `gosync is a fast, concurrent CLI utility for one-way synchronization of directories.
	It intelligently copies only new or modified files from source to destination.`,
	Run: func(cmd *cobra.Command, args []string) {
		// validate mandatory flags
		if opts.SourcePath == "" || opts.DestinationPath == "" {
			cmd.Help()
			fmt.Fprintln(os.Stderr, "\nError: --source and --dest are required arguments.")
			os.Exit(1) // Exit after error
		}

		// new Syncer instance
		syncerTool := syncer.NewSyncer(opts)

		fmt.Printf("-- Go Sync CLI ---\n")
		fmt.Printf("Source: %s\n", opts.SourcePath)
		fmt.Printf("Destination: %s\n", opts.DestinationPath)
		fmt.Printf("Workers: %d\n", syncerTool.Options.Workers)
		fmt.Printf("Dry Run: %t\n", opts.DryRun)
		fmt.Printf("Delete Extra Files: %t\n", opts.Delete)
		fmt.Printf("-------------------------------------------------- \n")

		startTime := time.Now()
		err := syncerTool.Start()
		elapsed := time.Since(startTime)

		// Handle result
		if err != nil {
			fmt.Fprintf(os.Stderr, "Synchronization failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\n Synchronization completed in %v\n", elapsed)

		os.Exit(0)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&opts.SourcePath, "source", "s", "", "The path to source directory. (Required)")
	rootCmd.Flags().StringVarP(&opts.DestinationPath, "dest", "d", "", "The path to destination directory. (Required)")

	rootCmd.Flags().BoolVar(&opts.Delete, "delete", false, "If present delete extra files and folders from destination.")
	rootCmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "If present what operation are performed without changing anything.")
	rootCmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "If present enable detailed logging of operation.")
	rootCmd.Flags().IntVar(&opts.Workers, "workers", runtime.NumCPU(), "Specifies the number of concurrent file copy workers.")
}
