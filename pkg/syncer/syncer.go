package syncer

type SyncOptions struct {
	SourcePath      string
	DestinationPath string
	DryRun          bool
	Delete          bool
	Verbose         bool
	Workers         int
}
