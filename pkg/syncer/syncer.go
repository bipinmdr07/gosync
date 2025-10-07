package syncer

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type SyncOptions struct {
	SourcePath      string
	DestinationPath string
	DryRun          bool
	Delete          bool
	Verbose         bool
	Workers         int
}

type Syncer struct {
	Options *SyncOptions
	wg      sync.WaitGroup
	fileOps chan string
	logger  zerolog.Logger
}

func NewSyncer(opts *SyncOptions) *Syncer {
	if opts.Workers == 0 {
		opts.Workers = runtime.NumCPU()
	}

	// Initialize Zerolog Console Writer for better readability in terminal
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	logger := zerolog.New(output).With().Timestamp().Logger()

	// Set log level based on flags
	if !opts.Verbose && !opts.DryRun {
		logger = logger.Level(zerolog.Disabled)
	} else if opts.Verbose {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}

	return &Syncer{
		Options: opts,
		fileOps: make(chan string),
		logger:  logger,
	}
}

func (s *Syncer) worker() {
	defer s.wg.Done()
	for srcPath := range s.fileOps {
		s.processFile(srcPath)
	}
}

// Handles the comparison and copying of a single file.
func (s *Syncer) processFile(srcPath string) {
	relPath, _ := filepath.Rel(s.Options.SourcePath, srcPath)
	destinationPath := filepath.Join(s.Options.DestinationPath, relPath)

	s.logger.Debug().Str("action", "CHECK_FILE").Str("path", relPath).Msg("File check started")

	// Check if source path exists
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		s.logger.Warn().Err(err).Str("path", srcPath).Msg("Could not stat source file")
	}

	// Check if destination exists and is up-to-date
	destInfo, err := os.Stat(destinationPath)
	if err == nil {
		// If destination file exists, compare modification times and sizes
		if !srcInfo.ModTime().After(destInfo.ModTime()) && srcInfo.Size() == destInfo.Size() {
			s.logger.Debug().Str("action", "SKIP_FILE").Str("path", relPath).Msg("File is up-to-date, skipping")
			return
		}
	} else if !os.IsNotExist(err) {
		s.logger.Warn().Str("path", destinationPath).Err(err).Msg("Could not stat destination file")
		return
	}

	s.logger.Info().Str("action", "COPY_FILE").Str("path", relPath).Str("destination", destinationPath).Msg("Copying file")
	// TODO: copy new or modified file
}

// TODO: Copy file function

// TODO: Delete extra files in destination function

func (s *Syncer) Start() error {
	// Check paths
	if s.Options.SourcePath == s.Options.DestinationPath {
		return fmt.Errorf("source and destination paths cannot be the same.")
	}

	// Start worker pool
	for i := 0; i < s.Options.Workers; i++ {
		s.wg.Add(1)
		go s.worker()
	}

	// Start file discovery and send jobs
	sourceFiles := make(map[string]bool)
	err := filepath.WalkDir(s.Options.SourcePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			s.logger.Error().Err(err).Str("path", path).Msg("Error walking source directory")
			return nil
		}

		relPath, _ := filepath.Rel(s.Options.SourcePath, path)
		fmt.Printf("- relPath: %s \n", relPath)
		if relPath == "." {
			return nil // Skip root
		}

		sourceFiles[relPath] = true

		if d.IsDir() {
			s.logger.Debug().Str("action", "CHECK_DIR").Str("path", relPath).Msg("Directory check started")
			return nil
		}

		s.fileOps <- path // Send full path to worker
		return nil
	})

	// Close channel and wait for workers to finish
	close(s.fileOps)
	s.wg.Wait()

	// TODO: Handle deletion of extra files in destination if s.Options.Delete is true

	return err // Return error from WalkDir if any
}
