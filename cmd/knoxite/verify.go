/*
 * knoxite
 *     Copyright (c) 2020, Fabian Siegel <fabians1999@gmail.com>
 *
 *   For license see LICENSE
 */

package main

import (
	"fmt"

	"github.com/knoxite/knoxite"
	"github.com/muesli/goprogressbar"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type VerifyOptions struct {
	Percentage int
}

var (
	verifyOpts = VerifyOptions{}

	verifyCmd = &cobra.Command{
		Use:   "verify [<volume> [<snapshot>]]",
		Short: "verify a repo, volume or snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return executeVerifyRepo(verifyOpts)
			} else if len(args) == 1 {
				return executeVerifyVolume(args[0], verifyOpts)
			} else if len(args) == 2 {
				return executeVerifySnapshot(args[0], args[1], verifyOpts)
			}
			return nil
		},
	}
)

func initVerifyFlags(f func() *pflag.FlagSet) {
	f().IntVar(&verifyOpts.Percentage, "percentage", 70, "How many archives to be checked between 0 and 100")
}

func init() {
	initVerifyFlags(verifyCmd.Flags)
	RootCmd.AddCommand(verifyCmd)
}

func executeVerifyRepo(opts VerifyOptions) error {
	errors := make([]error, 0)

	logger.Log(knoxite.Info, "Opening repository")
	repository, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, "Verifying repository and get knoxite progress")
	progress, err := knoxite.VerifyRepo(repository, opts.Percentage)
	if err != nil {
		errors = append(errors, err)
		return err
	}
	logger.Log(knoxite.Info, "Verified repository")

	logger.Log(knoxite.Info, "Initialising new goprogressbar for output")
	pb := &goprogressbar.ProgressBar{Total: 1000, Width: 40}
	lastPath := ""

	logger.Log(knoxite.Info, "Iterating over progress to print details")
	for p := range progress {
		if p.Error != nil {
			fmt.Println()
			errors = append(errors, p.Error)
		}

		pb.Total = int64(p.CurrentItemStats.Size)
		pb.Current = int64(p.CurrentItemStats.Transferred)
		pb.PrependText = fmt.Sprintf("%s / %s",
			knoxite.SizeToString(uint64(pb.Current)),
			knoxite.SizeToString(uint64(pb.Total)))

		if p.Path != lastPath {
			// We have just started restoring a new item
			if len(lastPath) > 0 {
				fmt.Println()
			}
			lastPath = p.Path
			pb.Text = p.Path
		}

		pb.LazyPrint()
	}
	fmt.Println()
	fmt.Printf("Verify done: %d errors\n", len(errors))

	logger.Log(knoxite.Info, "Printing verify repo output")
	logger.Log(knoxite.Info, "Verify repo command finished successfully")
	return nil
}

func executeVerifyVolume(volumeId string, opts VerifyOptions) error {
	errors := make([]error, 0)

	logger.Log(knoxite.Info, "Opening repository")
	repository, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, fmt.Sprintf("Verifying volume %s and get knoxite progress", volumeId))
	progress, err := knoxite.VerifyVolume(repository, volumeId, opts.Percentage)
	if err != nil {
		errors = append(errors, err)
		return err
	}
	logger.Log(knoxite.Info, "Verified volume")

	logger.Log(knoxite.Info, "Initialising new goprogressbar for output")
	pb := &goprogressbar.ProgressBar{Total: 1000, Width: 40}
	lastPath := ""

	logger.Log(knoxite.Info, "Iterating over progress to print details")
	for p := range progress {
		if p.Error != nil {
			fmt.Println()
			errors = append(errors, p.Error)
		}

		pb.Total = int64(p.CurrentItemStats.Size)
		pb.Current = int64(p.CurrentItemStats.Transferred)
		pb.PrependText = fmt.Sprintf("%s / %s",
			knoxite.SizeToString(uint64(pb.Current)),
			knoxite.SizeToString(uint64(pb.Total)))

		if p.Path != lastPath {
			// We have just started restoring a new item
			if len(lastPath) > 0 {
				fmt.Println()
			}
			lastPath = p.Path
			pb.Text = p.Path
		}

		pb.LazyPrint()
	}
	fmt.Println()
	fmt.Printf("Verify done: %d errors\n", len(errors))

	logger.Log(knoxite.Info, "Printing verify volume output")
	logger.Log(knoxite.Info, "Verify volume command finished successfully")
	return nil
}

func executeVerifySnapshot(volumeId string, snapshotId string, opts VerifyOptions) error {
	errors := make([]error, 0)

	logger.Log(knoxite.Info, "Opening repository")
	repository, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, fmt.Sprintf("Verifying snapshot %s and get knoxite progress", snapshotId))
	progress, err := knoxite.VerifySnapshot(repository, snapshotId, opts.Percentage)
	if err != nil {
		errors = append(errors, err)
		return err
	}
	logger.Log(knoxite.Info, "Verified snapshot")

	logger.Log(knoxite.Info, "Initialising new goprogressbar for output")
	pb := &goprogressbar.ProgressBar{Total: 1000, Width: 40}
	lastPath := ""

	logger.Log(knoxite.Info, "Iterating over progress to print details")
	for p := range progress {
		if p.Error != nil {
			fmt.Println()
			errors = append(errors, p.Error)
		}

		pb.Total = int64(p.CurrentItemStats.Size)
		pb.Current = int64(p.CurrentItemStats.Transferred)
		pb.PrependText = fmt.Sprintf("%s / %s",
			knoxite.SizeToString(uint64(pb.Current)),
			knoxite.SizeToString(uint64(pb.Total)))

		if p.Path != lastPath {
			// We have just started restoring a new item
			if len(lastPath) > 0 {
				fmt.Println()
			}
			lastPath = p.Path
			pb.Text = p.Path
		}

		pb.LazyPrint()
	}

	fmt.Println()
	fmt.Printf("Verify done: %d errors\n", len(errors))

	logger.Log(knoxite.Info, "Printing verify snapshot output")
	logger.Log(knoxite.Info, "Verify snapshot command finished successfully")
	return nil
}
