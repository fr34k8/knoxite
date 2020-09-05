/*
 * knoxite
 *     Copyright (c) 2016-2020, Christian Muehlhaeuser <muesli@gmail.com>
 *
 *   For license see LICENSE
 */

package main

import (
	"fmt"
	"path/filepath"

	shutdown "github.com/klauspost/shutdown2"
	"github.com/spf13/cobra"

	"github.com/knoxite/knoxite"
)

var (
	cloneOpts = StoreOptions{}

	cloneCmd = &cobra.Command{
		Use:   "clone <snapshot> <dir/file> [...]",
		Short: "clone a snapshot",
		Long:  `The clone command clones an existing snapshot and adds a file or directory`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("clone needs to know which snapshot to clone")
			}
			if len(args) < 2 {
				return fmt.Errorf("clone needs to know which files and/or directories to work on")
			}

			configureStoreOpts(cmd, &cloneOpts)
			return executeClone(args[0], args[1:], cloneOpts)
		},
	}
)

func init() {
	logger.Log(knoxite.Info, "Init store flags")
	initStoreFlags(cloneCmd.Flags)
	RootCmd.AddCommand(cloneCmd)
}

func executeClone(snapshotID string, args []string, opts StoreOptions) error {
	targets := []string{}
	logger.Log(knoxite.Info, "Collecting targets")
	for _, target := range args {
		if absTarget, err := filepath.Abs(target); err == nil {
			target = absTarget
		}
		targets = append(targets, target)
	}

	// acquire a shutdown lock. we don't want these next calls to be interrupted
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

	logger.Log(knoxite.Info, fmt.Sprintf("Finding snapshot %s", snapshotID))
	volume, s, err := repository.FindSnapshot(snapshotID)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, fmt.Sprintf("Found snapshot %s", s.Description))

	logger.Log(knoxite.Info, "Cloning snapshot")
	snapshot, err := s.Clone()
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, fmt.Sprintf("Cloned snapshot. New snapshot: ID: %s, "+
		"Description: %s.", snapshot.ID, snapshot.Description))

	logger.Log(knoxite.Info, "Opening chunk index")
	chunkIndex, err := knoxite.OpenChunkIndex(&repository)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened chunk index")

	lock()
	logger.Log(knoxite.Info, "Released shutdown lock")

	logger.Log(knoxite.Info, fmt.Sprintf("Storing cloned snapshot %s", snapshot.ID))
	err = store(&repository, &chunkIndex, snapshot, targets, opts)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, fmt.Sprintf("Stored clone %s of snapshot %s", snapshot.ID, s.ID))

	// we don't want these next calls to be interrupted
	logger.Log(knoxite.Info, "Acquiring shutdown lock")
	lock = shutdown.Lock()
	if lock == nil {
		return nil
	}
	logger.Log(knoxite.Info, "Acquired and locked shutdown lock")

	defer lock()
	defer logger.Log(knoxite.Info, "Shutdown lock released")

	logger.Log(knoxite.Info, fmt.Sprintf("Saving snapshot %s", snapshot.ID))
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
	logger.Log(knoxite.Info, "Clone command finished successfully")
	return nil
}
