package syncer

import (
	"fmt"
	"io"
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
	s.copyFile(srcPath, destinationPath, srcInfo)
}

// Function to copy files from source to destination, creating directories as needed.
func (s *Syncer) copyFile(srcPath, destinationPath string, srcInfo os.FileInfo) {
	relPath, _ := filepath.Rel(s.Options.SourcePath, srcPath)
	logEvent := s.logger.Info().Str("action", "COPY").Str("path", relPath)

	if s.Options.DryRun {
		logEvent.Msg("DRY_RUN: Would copy file")
		return
	}

	// Create parent directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(destinationPath), os.ModePerm); err != nil {
		s.logger.Error().Err(err).Str("path", destinationPath).Msg("Failed to create directories")
		return
	}

	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		s.logger.Error().Err(err).Str("path", srcPath).Msg("Error opening source file")
		return
	}
	defer srcFile.Close()

	// Create/overwrite destination file
	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		s.logger.Error().Err(err).Str("path", destinationPath).Msg("Error creating destination file")
		return
	}
	defer destinationFile.Close()

	// Copy file contents
	if _, err := io.Copy(destinationFile, srcFile); err != nil {
		s.logger.Error().Err(err).Str("path", destinationPath).Msg("Error copying file contents")
		return
	}

	// Sync and Preserve modification time
	destinationFile.Sync()
	if err := os.Chtimes(destinationPath, time.Now(), srcInfo.ModTime()); err != nil {
		s.logger.Warn().Err(err).Str("path", destinationPath).Msg("Error preserving modification time")
	}

	// Set file permissions for source
	if err := os.Chmod(destinationPath, srcInfo.Mode()); err != nil {
		s.logger.Warn().Err(err).Str("path", destinationPath).Msg("Error setting file permissions")
	}

	logEvent.Msg("File copied successfully")
}

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
