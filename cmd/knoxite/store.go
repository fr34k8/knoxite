/*
 * knoxite
 *     Copyright (c) 2016-2020, Christian Muehlhaeuser <muesli@gmail.com>
 *
 *   For license see LICENSE
 */

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	humanize "github.com/dustin/go-humanize"
	shutdown "github.com/klauspost/shutdown2"
	"github.com/muesli/goprogressbar"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/knoxite/knoxite"
	"github.com/knoxite/knoxite/cmd/knoxite/utils"
)

// Error declarations
var (
	ErrRedundancyAmount = errors.New("failure tolerance can't be equal or higher as the number of storage backends")
)

// StoreOptions holds all the options that can be set for the 'store' command.
type StoreOptions struct {
	Description      string
	Compression      string
	Encryption       string
	FailureTolerance uint
	Excludes         []string
}

var (
	storeOpts = StoreOptions{}

	storeCmd = &cobra.Command{
		Use:   "store <volume> <dir/file> [...]",
		Short: "store files/directories",
		Long:  `The store command creates a snapshot of a file or directory`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("store needs to know which volume to create a snapshot in")
			}
			if len(args) < 2 {
				return fmt.Errorf("store needs to know which files and/or directories to work on")
			}

			configureStoreOpts(cmd, &storeOpts)
			return executeStore(args[0], args[1:], storeOpts)
		},
	}
)

// configureStoreOpts will compare the settings from the configuration file and
// the user set command line flags.
// Values set via the command line flags will overwrite settings stored in the
// configuration file.
func configureStoreOpts(cmd *cobra.Command, opts *StoreOptions) {
	if rep, ok := cfg.Repositories[globalOpts.Repo]; ok {
		if !cmd.Flags().Changed("compression") {
			opts.Compression = rep.Compression
		}
		if !cmd.Flags().Changed("encryption") {
			opts.Encryption = rep.Encryption
		}
		if !cmd.Flags().Changed("tolerance") {
			opts.FailureTolerance = rep.Tolerance
		}
		if !cmd.Flags().Changed("excludes") {
			opts.Excludes = rep.StoreExcludes
		}
	}
}

func initStoreFlags(f func() *pflag.FlagSet) {
	f().StringVarP(&storeOpts.Description, "desc", "d", "", "a description or comment for this volume")
	f().StringVarP(&storeOpts.Compression, "compression", "c", "", "compression algo to use: none (default), flate, gzip, lzma, zlib, zstd")
	f().StringVarP(&storeOpts.Encryption, "encryption", "e", "", "encryption algo to use: aes (default), none")
	f().UintVarP(&storeOpts.FailureTolerance, "tolerance", "t", 0, "failure tolerance against n backend failures")
	f().StringArrayVarP(&storeOpts.Excludes, "excludes", "x", []string{}, "list of excludes")
}

func init() {
	initStoreFlags(storeCmd.Flags)
	RootCmd.AddCommand(storeCmd)
}

func store(repository *knoxite.Repository, chunkIndex *knoxite.ChunkIndex, snapshot *knoxite.Snapshot, targets []string, opts StoreOptions) error {
	// we want to be notified during the first phase of a shutdown
	logger.Log(knoxite.Info, "Acquiring shutdown notifier")
	cancel := shutdown.First()

	logger.Log(knoxite.Info, "Getting rooted path name corresponding to the current directory")
	wd, gerr := os.Getwd()
	if gerr != nil {
		return gerr
	}
	logger.Log(knoxite.Info, fmt.Sprintf("Rooted path: %s", wd))

	if len(repository.BackendManager().Backends)-int(opts.FailureTolerance) <= 0 {
		return ErrRedundancyAmount
	}

	logger.Log(knoxite.Info, "Get compression type from options")
	compression, err := utils.CompressionTypeFromString(opts.Compression)
	if err != nil {
		return err
	}

	logger.Log(knoxite.Info, "Get encryption type from options")
	encryption, err := utils.EncryptionTypeFromString(opts.Encryption)
	if err != nil {
		return err
	}

	tol := uint(len(repository.BackendManager().Backends) - int(opts.FailureTolerance))

	startTime := time.Now()

	logger.Log(knoxite.Info, "Adding snapshot and getting progress")
	progress := snapshot.Add(wd, targets, opts.Excludes, *repository, chunkIndex,
		compression, encryption,
		tol, opts.FailureTolerance)

	logger.Log(knoxite.Info, "Initialising new goprogressbar for output")
	fileProgressBar := &goprogressbar.ProgressBar{Width: 40}
	overallProgressBar := &goprogressbar.ProgressBar{
		Text:  fmt.Sprintf("%d of %d total", 0, 0),
		Width: 60,
		PrependTextFunc: func(p *goprogressbar.ProgressBar) string {
			return fmt.Sprintf("%s/s",
				knoxite.SizeToString(uint64(float64(p.Current)/time.Since(startTime).Seconds())))
		},
	}

	pb := goprogressbar.MultiProgressBar{}
	pb.AddProgressBar(fileProgressBar)
	pb.AddProgressBar(overallProgressBar)

	lastPath := ""
	items := int64(1)

	logger.Log(knoxite.Info, "Iterating over progress to print details")
	for p := range progress {
		select {
		case n := <-cancel:
			logger.Log(knoxite.Info, "Operation got cancelled. Aborting.")

			fmt.Println("Aborting...")
			close(n)
			return nil

		default:
			if p.Error != nil {
				fmt.Println()
				return p.Error
			}
			if p.Path != lastPath && lastPath != "" {
				items++
				fmt.Println()
			}
			fileProgressBar.Total = int64(p.CurrentItemStats.Size)
			fileProgressBar.Current = int64(p.CurrentItemStats.Transferred)
			fileProgressBar.PrependText = fmt.Sprintf("%s  %s/s",
				knoxite.SizeToString(uint64(fileProgressBar.Current)),
				knoxite.SizeToString(p.TransferSpeed()))

			overallProgressBar.Total = int64(p.TotalStatistics.Size)
			overallProgressBar.Current = int64(p.TotalStatistics.Transferred)
			overallProgressBar.Text = fmt.Sprintf("%s / %s (%s of %s)",
				knoxite.SizeToString(uint64(overallProgressBar.Current)),
				knoxite.SizeToString(uint64(overallProgressBar.Total)),
				humanize.Comma(items),
				humanize.Comma(int64(p.TotalStatistics.Files+p.TotalStatistics.Dirs+p.TotalStatistics.SymLinks)))

			if p.Path != lastPath {
				lastPath = p.Path
				fileProgressBar.Text = p.Path
			}

			logger.Log(knoxite.Debug, "Printing progressbar")
			pb.LazyPrint()
		}
	}

	fmt.Printf("\nSnapshot %s created: %s\n", snapshot.ID, snapshot.Stats.String())
	logger.Log(knoxite.Info, "Store finished successfully")

	return nil
}

func executeStore(volumeID string, args []string, opts StoreOptions) error {
	targets := []string{}

	logger.Log(knoxite.Info, "Collecting targets")
	for _, target := range args {
		if absTarget, err := filepath.Abs(target); err == nil {
			target = absTarget
		}
		targets = append(targets, target)
		logger.Log(knoxite.Info, "Collected targets")
	}

	// we don't want these next calls to be interrupted
	logger.Log(knoxite.Info, "Acquiring shutdown lock")
	lock := shutdown.Lock()
	if lock == nil {
		return nil
	}
	logger.Log(knoxite.Info, "Acquired and locked shutdown lock")

	logger.Log(knoxite.Info, "Opening repository")
	repository, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, fmt.Sprintf("Finding volume %s", volumeID))
	volume, err := repository.FindVolume(volumeID)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Found volume")

	logger.Log(knoxite.Info, fmt.Sprintf("Creating new snapshot: %s", opts.Description))
	snapshot, err := knoxite.NewSnapshot(opts.Description)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Created snapshot")

	logger.Log(knoxite.Info, "Opening chunk index")
	chunkIndex, err := knoxite.OpenChunkIndex(&repository)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened chunk index")

	// release the shutdown lock
	lock()
	logger.Log(knoxite.Info, "Shutdown lock released")

	logger.Log(knoxite.Info, fmt.Sprintf("Storing snapshot %s", snapshot.ID))
	err = store(&repository, &chunkIndex, snapshot, targets, opts)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Stored snapshot")

	// we don't want these next calls to be interrupted
	logger.Log(knoxite.Info, "Acquiring another shutdown lock")
	lock = shutdown.Lock()
	if lock == nil {
		return nil
	}
	defer lock()
	defer logger.Log(knoxite.Info, "Shutdown lock released")

	logger.Log(knoxite.Info, "Saving snapshot")
	err = snapshot.Save(&repository)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Saved snapshot")

	logger.Log(knoxite.Info, fmt.Sprintf("Adding snapshot to volume %s", volume.ID))
	err = volume.AddSnapshot(snapshot.ID)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Added snapshot to volume")

	logger.Log(knoxite.Info, "Saving chunk index")
	err = chunkIndex.Save(&repository)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Saved chunk index")

	logger.Log(knoxite.Info, "Saving repository")
	err = repository.Save()
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Saved repository")
	logger.Log(knoxite.Info, "Store command finished successfully")

	return nil
}
